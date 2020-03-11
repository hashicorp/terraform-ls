package langserver

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	rpcServer "github.com/creachadair/jrpc2/server"
	"github.com/hashicorp/terraform-ls/langserver/srvctl"
)

type langServer struct {
	ctx        context.Context
	hp         srvctl.HandlerProvider
	logger     *log.Logger
	srvOptions *jrpc2.ServerOptions

	srvCtl   srvctl.ServerController
	stopFunc context.CancelFunc
}

func NewLangServer(srvCtx context.Context, hp srvctl.HandlerProvider) *langServer {
	opts := &jrpc2.ServerOptions{
		AllowPush: true,
	}

	srvCtx, stopFunc := context.WithCancel(srvCtx)

	return &langServer{
		ctx:        srvCtx,
		hp:         hp,
		logger:     log.New(ioutil.Discard, "", 0),
		srvOptions: opts,
		srvCtl:     srvctl.NewServerController(),
		stopFunc:   stopFunc,
	}
}

func (ls *langServer) SetLogger(logger *log.Logger) {
	ls.srvOptions.Logger = logger
	ls.srvOptions.RPCLog = &rpcLogger{logger}
	ls.hp.SetLogger(logger)
	ls.logger = logger
}

func (ls *langServer) start(reader io.Reader, writer io.WriteCloser) *jrpc2.Server {
	err := ls.srvCtl.Prepare()
	if err != nil {
		ls.logger.Printf("Unable to prepare server: %s", err)
		ls.stopFunc()
		return nil
	}

	ch := channel.LSP(reader, writer)

	return jrpc2.NewServer(ls.hp.Handlers(ls.srvCtl), ls.srvOptions).Start(ch)
}

func (ls *langServer) StartAndWait(reader io.Reader, writer io.WriteCloser) {
	srv := ls.start(reader, writer)
	go func() {
		ls.logger.Println("Starting server ...")
		err := srv.Wait()
		if err != nil {
			ls.logger.Printf("Server failed to start: %s", err)
			ls.stopFunc()
			return
		}
	}()

	select {
	case <-ls.ctx.Done():
		ls.logger.Println("Stopping server ...")
		srv.Stop()
	}
}

func (ls *langServer) StartTCP(address string) error {
	err := ls.srvCtl.Prepare()
	if err != nil {
		ls.logger.Printf("Unable to prepare server: %s", err)
		ls.stopFunc()
	}

	ls.logger.Printf("Starting TCP server at %q ...", address)
	lst, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("TCP Server failed to start: %s", err)
	}
	ls.logger.Printf("TCP server running at %q", lst.Addr())

	go func() {
		ls.logger.Println("Starting loop server ...")
		err = rpcServer.Loop(lst, ls.hp.Handlers(ls.srvCtl), &rpcServer.LoopOptions{
			Framing:       channel.LSP,
			ServerOptions: ls.srvOptions,
		})
		if err != nil {
			ls.logger.Printf("Loop server failed to start: %s", err)
			ls.stopFunc()
			return
		}
	}()

	select {
	case <-ls.ctx.Done():
		ls.logger.Println("Shutting down server ...")
		err := lst.Close()
		if err != nil {
			ls.logger.Printf("TCP Server failed to shutdown: %s", err)
			return ls.ctx.Err()
		}
	}

	return nil
}
