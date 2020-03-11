package srvctl

import (
	"log"

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
	Handlers(ServerController) jrpc2.Assigner
	SetLogger(*log.Logger)
}
