package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
	"github.com/creachadair/jrpc2/server"
	"github.com/radeksimko/terraform-ls/internal/filesystem"
)

const ctxFs = "ctxFilesystem"

type langServer struct {
	handlerMap handler.Map
	logger     *log.Logger
	srvOptions *jrpc2.ServerOptions
}

func NewLangServer(logger *log.Logger) *langServer {
	m := handler.Map{
		"initialize":              handler.New(Initialize),
		"textDocument/completion": handler.New(TextDocumentComplete),
		"textDocument/didChange":  handler.New(TextDocumentDidChange),
		"textDocument/didOpen":    handler.New(TextDocumentDidOpen),
		"textDocument/didClose":   handler.New(TextDocumentDidClose),
		"exit":                    handler.New(Exit),
		"shutdown":                handler.New(Shutdown),
		"$/cancelRequest":         handler.New(CancelRequest),
	}

	fs := filesystem.NewFilesystem()

	opts := &jrpc2.ServerOptions{
		AllowPush: true,
		Logger:    logger,
		DecodeContext: func(ctx context.Context, _ string, msg json.RawMessage) (context.Context, json.RawMessage, error) {
			return context.WithValue(ctx, ctxFs, fs), msg, nil
		},
	}

	return &langServer{
		handlerMap: m,
		logger:     logger,
		srvOptions: opts,
	}
}

func (ls *langServer) Start(ctx context.Context, reader io.Reader, writer io.WriteCloser) {
	ch := LspFraming(ls.logger)(reader, writer)

	srv := jrpc2.NewServer(ls.handlerMap, ls.srvOptions).Start(ch)
	ctx, cancelFunc := context.WithCancel(ctx)

	go func() {
		ls.logger.Println("Starting server ...")
		err := srv.Wait()
		if err != nil {
			ls.logger.Printf("Server failed to start: %s", err)
			cancelFunc()
			return
		}
	}()

	select {
	case <-ctx.Done():
		srv.Stop()
	}
}

func (ls *langServer) StartTCP(ctx context.Context, address string) error {
	ls.logger.Printf("Starting TCP server at %q ...", address)
	lst, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("TCP Server failed to start: %s", err)
	}
	ls.logger.Printf("TCP server running at %q", lst.Addr())

	ctx, cancelFunc := context.WithCancel(ctx)

	go func() {
		ls.logger.Println("Starting loop server ...")
		err = server.Loop(lst, ls.handlerMap, &server.LoopOptions{
			Framing:       LspFraming(ls.logger),
			ServerOptions: ls.srvOptions,
		})
		if err != nil {
			ls.logger.Printf("Loop server failed to start: %s", err)
			cancelFunc()
			return
		}
	}()

	select {
	case <-ctx.Done():
		ls.logger.Println("Shutting down server ...")
		err := lst.Close()
		if err != nil {
			ls.logger.Printf("TCP Server failed to shutdown: %s", err)
			return ctx.Err()
		}
	}

	return nil
}
