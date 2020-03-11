package langserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/langserver/srvctl"
)

type langServerMock struct {
	srv         *langServer
	srvStopFunc context.CancelFunc
	srvStdin    io.Reader
	srvStdout   io.WriteCloser

	client       *jrpc2.Client
	clientStdin  io.Reader
	clientStdout io.WriteCloser
}

func NewLangServerMock(t *testing.T, hp srvctl.HandlerProvider) *langServerMock {
	ctx, stopFunc := context.WithCancel(context.Background())

	srv := NewLangServer(ctx, hp)
	srv.SetLogger(testLogger(os.Stdout, "[SERVER] "))

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	return &langServerMock{
		srv:          srv,
		srvStopFunc:  stopFunc,
		srvStdin:     stdinReader,
		srvStdout:    stdoutWriter,
		clientStdin:  stdoutReader,
		clientStdout: stdinWriter,
	}
}

func (lsm *langServerMock) Start(t *testing.T) context.CancelFunc {
	rpcSrv := lsm.srv.start(lsm.srvStdin, lsm.srvStdout)
	go func() {
		rpcSrv.Wait()
	}()

	clientCh := channel.LSP(lsm.clientStdin, lsm.clientStdout)
	lsm.client = jrpc2.NewClient(clientCh, &jrpc2.ClientOptions{
		Logger: testLogger(os.Stdout, "[CLIENT] "),
	})

	return lsm.srvStopFunc
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
