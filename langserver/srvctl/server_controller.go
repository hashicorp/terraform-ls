package srvctl

import (
	"log"
	"context"

	"github.com/creachadair/jrpc2"
)

type ServerController interface {
	CheckInitializationIsConfirmed() error
	ConfirmInitialization(*jrpc2.Request) error
	Exit() error
	Initialize(*jrpc2.Request) error
	Prepare() error
	Shutdown(*jrpc2.Request) error
}

type HandlerProvider interface {
	Handlers(context.Context, ServerController) jrpc2.Assigner
	SetLogger(*log.Logger)
}
