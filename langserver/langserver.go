package langserver

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/creachadair/jrpc2"
	rpcServer "github.com/creachadair/jrpc2/server"
)

type langServer struct {
	ctx        context.Context
	assigner   jrpc2.Assigner
	logger     *log.Logger
	srvOptions *jrpc2.ServerOptions

	server   *server
	stopFunc context.CancelFunc
}

func NewLangServer(srvCtx context.Context, logger *log.Logger) *langServer {
	srv := newServer()

	opts := &jrpc2.ServerOptions{
		AllowPush: true,
		Logger:    logger,
	}

	srvCtx, stopFunc := context.WithCancel(srvCtx)
	hm := &handlerMap{logger: logger, srv: srv, srvStopFunc: stopFunc}

	return &langServer{
		ctx:        srvCtx,
		assigner:   hm.Map(),
		logger:     logger,
		srvOptions: opts,
		server:     srv,
		stopFunc:   stopFunc,
	}
}

func (ls *langServer) Start(reader io.Reader, writer io.WriteCloser) {
	err := ls.server.Prepare()
	if err != nil {
		ls.logger.Printf("Unable to prepare server: %s", err)
		ls.stopFunc()
	}

	ch := LspFraming(ls.logger)(reader, writer)

	srv := jrpc2.NewServer(ls.assigner, ls.srvOptions).Start(ch)

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
	err := ls.server.Prepare()
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
		err = rpcServer.Loop(lst, ls.assigner, &rpcServer.LoopOptions{
			Framing:       LspFraming(ls.logger),
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
