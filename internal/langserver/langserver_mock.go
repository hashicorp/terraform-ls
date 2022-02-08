package langserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
)

type langServerMock struct {
	srv    *langServer
	logger *log.Logger

	// rpcSrv is set when server starts and allows Stop() to stop it after testing is finished
	rpcSrv *singleServer

	srvStopFunc    context.CancelFunc
	stopFuncCalled bool

	srvStdin  io.Reader
	srvStdout io.WriteCloser

	client       *jrpc2.Client
	clientStdin  io.Reader
	clientStdout io.WriteCloser
}

func NewLangServerMock(t *testing.T, sf session.SessionFactory) *langServerMock {
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	logger := discardLogger()
	if testing.Verbose() {
		logger = testLogger(os.Stdout, "")
	}

	srvCtx, stopFunc := context.WithCancel(context.Background())
	lsm := &langServerMock{
		logger:       logger,
		srvStopFunc:  stopFunc,
		srvStdin:     stdinReader,
		srvStdout:    stdoutWriter,
		clientStdin:  stdoutReader,
		clientStdout: stdinWriter,
	}

	lsm.srv = NewLangServer(srvCtx, func(srvCtx context.Context) session.Session {
		return sf(srvCtx)
	})
	if testing.Verbose() {
		lsm.srv.SetLogger(testLogger(os.Stdout, "[SERVER] "))
	}

	return lsm
}

func (lsm *langServerMock) Stop() {
	lsm.logger.Println("Stopping mock server ...")
	lsm.rpcSrv.Stop()
	lsm.stopFuncCalled = true
}

func (lsm *langServerMock) StopFuncCalled() bool {
	return lsm.stopFuncCalled
}

// Start is more or less duplicate of langServer.StartAndWait
// except that this one doesn't wait
//
// TODO: Explore whether we could leverage jrpc2's server.Local
func (lsm *langServerMock) Start(t *testing.T) context.CancelFunc {
	lsm.logger.Println("Starting mock server ...")

	srv, err := lsm.srv.startServer(lsm.srvStdin, lsm.srvStdout)
	if err != nil {
		t.Fatal(err)
	}

	lsm.rpcSrv = srv

	go func() {
		lsm.rpcSrv.Wait()
	}()

	clientCh := channel.LSP(lsm.clientStdin, lsm.clientStdout)
	opts := &jrpc2.ClientOptions{
		OnCallback: func(c context.Context, r *jrpc2.Request) (interface{}, error) {
			if r.Method() == "window/workDoneProgress/create" {
				return nil, nil
			}
			return nil, fmt.Errorf("method not implemented: %q", r.Method())
		},
	}
	if testing.Verbose() {
		opts.Logger = jrpc2.StdLogger(testLogger(os.Stdout, "[CLIENT] "))
	}
	lsm.client = jrpc2.NewClient(clientCh, opts)

	return lsm.Stop
}

func (lsm *langServerMock) CloseClientStdout(t *testing.T) {
	err := lsm.clientStdout.Close()
	if err != nil {
		t.Fatal(err)
	}
}

type CallRequest struct {
	Method    string
	ReqParams string
}

func (lsm *langServerMock) Call(t *testing.T, cr *CallRequest) *rawResponse {
	rsp, err := lsm.client.Call(context.Background(), cr.Method, json.RawMessage(cr.ReqParams))
	if err != nil {
		t.Fatal(err)
	}
	b, err := rsp.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	r := &rawResponse{}
	err = r.UnmarshalJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (lsm *langServerMock) CallAndExpectResponse(t *testing.T, cr *CallRequest, expectRaw string) {
	rsp := lsm.Call(t, cr)

	// Compacting is necessary because we retain params as json.RawMessage
	// in rawResponse, which holds formatted bytes that may not match
	// due to whitespaces
	compactedRaw := bytes.NewBuffer([]byte{})
	err := json.Compact(compactedRaw, []byte(expectRaw))
	if err != nil {
		t.Fatal(err)
	}

	expectedRsp := &rawResponse{}
	err = expectedRsp.UnmarshalJSON(compactedRaw.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expectedRsp, rsp); diff != "" {
		t.Fatalf("%q response doesn't match.\n%s",
			cr.Method, diff)
	}
}

func (lsm *langServerMock) CallAndExpectError(t *testing.T, cr *CallRequest, expectErr error) {
	_, err := lsm.client.Call(context.Background(), cr.Method, json.RawMessage(cr.ReqParams))
	if err == nil {
		t.Fatalf("expected error: %s", expectErr.Error())
	}

	if !errors.Is(expectErr, err) {
		t.Fatalf("%q error doesn't match.\nexpected: %#v\ngiven: %#v\n",
			cr.Method, expectErr, err)
	}
}

func (lsm *langServerMock) Notify(t *testing.T, cr *CallRequest) {
	err := lsm.client.Notify(context.Background(), cr.Method, json.RawMessage(cr.ReqParams))

	// This is to account for the fact that
	// notifications are asynchronous in nature per LSP spec.
	//
	// We assume the server under test has no other notifications
	// to process and the method is quick to execute.
	//
	// TODO: We may need to re-evaluate this hack later and check
	// if the server could be turned into sync mode somehow
	time.Sleep(1 * time.Millisecond)

	if err != nil {
		t.Fatal(err)
	}
}

// rawResponse is a copy of jrpc2.jresponse
// to enable accurate comparison of responses
type rawResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Error   *jrpc2.Error    `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`

	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

func (r *rawResponse) UnmarshalJSON(b []byte) error {
	type t rawResponse
	var resp t

	err := json.Unmarshal(b, &resp)
	if err != nil {
		return err
	}

	*r = *(*rawResponse)(&resp)
	return nil
}

func testLogger(w io.Writer, prefix string) *log.Logger {
	return log.New(w, prefix, log.LstdFlags|log.Lshortfile)
}

func discardLogger() *log.Logger {
	return log.New(ioutil.Discard, "", 0)
}
