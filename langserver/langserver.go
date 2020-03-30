package langserver

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/server"
	"github.com/hashicorp/terraform-ls/langserver/svcctl"
)

type langServer struct {
	srvCtx     context.Context
	logger     *log.Logger
	srvOptions *jrpc2.ServerOptions
	sf         svcctl.ServiceFactory
}

func NewLangServer(srvCtx context.Context, sf svcctl.ServiceFactory) *langServer {
	opts := &jrpc2.ServerOptions{
		AllowPush: true,
	}

	return &langServer{
		srvCtx:     srvCtx,
		logger:     log.New(ioutil.Discard, "", 0),
		srvOptions: opts,
		sf:         sf,
	}
}

func (ls *langServer) SetLogger(logger *log.Logger) {
	ls.srvOptions.Logger = logger
	ls.srvOptions.RPCLog = &rpcLogger{logger}
	ls.logger = logger
}

func (ls *langServer) newService() server.Service {
	svc := ls.sf(ls.srvCtx)
	svc.SetLogger(ls.logger)
	return svc
}

func (ls *langServer) startServer(reader io.Reader, writer io.WriteCloser) (*singleServer, error) {
	srv, err := Server(ls.newService(), ls.srvOptions)
	if err != nil {
		return nil, err
	}
	srv.Start(channel.LSP(reader, writer))

	return srv, nil
}

func (ls *langServer) StartAndWait(reader io.Reader, writer io.WriteCloser) error {
	srv, err := ls.startServer(reader, writer)
	if err != nil {
		return err
	}
	ls.logger.Printf("Starting server (pid %d) ...", os.Getpid())

	// Wrap waiter with a context so that we can cancel it here
	// after the service is cancelled (and srv.Wait returns)
	ctx, cancelFunc := context.WithCancel(ls.srvCtx)
	go func() {
		srv.Wait()
		cancelFunc()
	}()

	select {
	case <-ctx.Done():
		ls.logger.Printf("Stopping server (pid %d) ...", os.Getpid())
		srv.Stop()
	}

	ls.logger.Printf("Server (pid %d) stopped.", os.Getpid())
	return nil
}

func (ls *langServer) StartTCP(address string) error {
	ls.logger.Printf("Starting TCP server (pid %d) at %q ...",
		os.Getpid(), address)
	lst, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("TCP Server failed to start: %s", err)
	}
	ls.logger.Printf("TCP server running at %q", lst.Addr())

	go func() {
		ls.logger.Println("Starting loop server ...")
		err = server.Loop(lst, ls.newService, &server.LoopOptions{
			Framing:       channel.LSP,
			ServerOptions: ls.srvOptions,
		})
		if err != nil {
			ls.logger.Printf("Loop server failed to start: %s", err)
		}
	}()

	select {
	case <-ls.srvCtx.Done():
		ls.logger.Println("Shutting down TCP server ...")
		err = lst.Close()
		if err != nil {
			ls.logger.Printf("TCP server failed to shutdown: %s", err)
			return err
		}
	}

	return nil
}

// singleServer is a wrapper around jrpc2.NewServer providing support
// for server.Service (Assigner/Finish interface)
type singleServer struct {
	srv        *jrpc2.Server
	finishFunc func(jrpc2.ServerStatus)
}

func Server(svc server.Service, opts *jrpc2.ServerOptions) (*singleServer, error) {
	assigner, err := svc.Assigner()
	if err != nil {
		return nil, err
	}

	return &singleServer{
		srv:        jrpc2.NewServer(assigner, opts),
		finishFunc: svc.Finish,
	}, nil
}

func (ss *singleServer) Start(ch channel.Channel) {
	ss.srv = ss.srv.Start(ch)
}

func (ss *singleServer) StartAndWait(ch channel.Channel) {
	ss.Start(ch)
	ss.Wait()
}

func (ss *singleServer) Wait() {
	status := ss.srv.WaitStatus()
	ss.finishFunc(status)
}

func (ss *singleServer) Stop() {
	ss.srv.Stop()
}
