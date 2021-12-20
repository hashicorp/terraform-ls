package langserver

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/server"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
)

type langServer struct {
	srvCtx     context.Context
	logger     *log.Logger
	srvOptions *jrpc2.ServerOptions
	newSession session.SessionFactory
}

type ctxReqConcurrency struct{}

func NewLangServer(srvCtx context.Context, sf session.SessionFactory) *langServer {
	concurrency, ok := requestConcurrencyFromCtx(srvCtx)
	if !ok {
		concurrency = DefaultConcurrency()
	}

	opts := &jrpc2.ServerOptions{
		AllowPush:   true,
		Concurrency: concurrency,
	}

	return &langServer{
		srvCtx:     srvCtx,
		logger:     log.New(ioutil.Discard, "", 0),
		srvOptions: opts,
		newSession: sf,
	}
}

func WithRequestConcurrency(parent context.Context, concurrency int) context.Context {
	return context.WithValue(parent, ctxReqConcurrency{}, concurrency)
}

func requestConcurrencyFromCtx(ctx context.Context) (int, bool) {
	c, ok := ctx.Value(ctxReqConcurrency{}).(int)
	return c, ok
}

func DefaultConcurrency() int {
	cpu := runtime.NumCPU()
	// Cap concurrency on powerful machines
	// to leave some capacity for module ops
	// and other application
	if cpu >= 4 {
		return cpu / 2
	}
	return cpu
}

func (ls *langServer) SetLogger(logger *log.Logger) {
	ls.srvOptions.Logger = jrpc2.StdLogger(logger)
	ls.srvOptions.RPCLog = &rpcLogger{logger}
	ls.logger = logger
}

func (ls *langServer) newService() server.Service {
	svc := ls.newSession(ls.srvCtx)
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
	ls.logger.Printf("Starting server (pid %d; concurrency: %d) ...",
		os.Getpid(), ls.srvOptions.Concurrency)

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
	ls.logger.Printf("Starting TCP server (pid %d; concurrency: %d) at %q ...",
		os.Getpid(), ls.srvOptions.Concurrency, address)
	lst, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("TCP Server failed to start: %s", err)
	}
	ls.logger.Printf("TCP server running at %q", lst.Addr())

	accepter := server.NetAccepter(lst, channel.LSP)

	go func() {
		ls.logger.Println("Starting loop server ...")
		err = server.Loop(context.TODO(), accepter, ls.newService, &server.LoopOptions{
			ServerOptions: ls.srvOptions,
		})
		if err != nil {
			ls.logger.Printf("Loop server failed to start: %s", err)
		}
	}()

	select {
	case <-ls.srvCtx.Done():
		ls.logger.Printf("Stopping TCP server (pid %d) ...", os.Getpid())
		err = lst.Close()
		if err != nil {
			ls.logger.Printf("TCP server (pid %d) failed to stop: %s", os.Getpid(), err)
			return err
		}
	}

	ls.logger.Printf("TCP server (pid %d) stopped.", os.Getpid())
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
		srv: jrpc2.NewServer(assigner, opts),
		finishFunc: func(status jrpc2.ServerStatus) {
			svc.Finish(assigner, status)
		},
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
