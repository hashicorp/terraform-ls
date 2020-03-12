package srvctl

import (
	"context"
	"fmt"
	"time"

	"github.com/creachadair/jrpc2"
)

// serverState represents state of the language server
// with respect to the LSP
type serverState int

const (
	// Before server starts
	stateEmpty serverState = -1
	// After server starts, before any request
	statePrepared serverState = 0
	// After "initialize", before "initialized"
	stateInitializedUnconfirmed serverState = 1
	// After "initialized"
	stateInitializedConfirmed serverState = 2
	// After "shutdown"
	stateDown serverState = 3
)

func (ss serverState) String() string {
	switch ss {
	case stateEmpty:
		return "<empty>"
	case statePrepared:
		return "prepared"
	case stateInitializedUnconfirmed:
		return "initialized (unconfirmed)"
	case stateInitializedConfirmed:
		return "initialized (confirmed)"
	case stateDown:
		return "down"
	}
	return "<unknown>"
}

type server struct {
	initializeReq     *jrpc2.Request
	initializeReqTime time.Time

	initializedReq     *jrpc2.Request
	initializedReqTime time.Time

	downReq     *jrpc2.Request
	downReqTime time.Time

	state    serverState
	exitFunc context.CancelFunc
}

func (srv *server) isPrepared() bool {
	return srv.state == statePrepared
}

func (srv *server) Prepare() error {
	if srv.state != stateEmpty {
		return &unexpectedSrvState{
			ExpectedState: stateInitializedConfirmed,
			CurrentState:  srv.state,
		}
	}

	srv.state = statePrepared

	return nil
}

func (srv *server) IsInitializedUnconfirmed() bool {
	return srv.state == stateInitializedUnconfirmed
}

func (srv *server) Initialize(req *jrpc2.Request) error {
	if srv.state != statePrepared {
		if srv.IsInitializedUnconfirmed() {
			return srvAlreadyInitializedErr(srv.initializeReq.ID())
		}
		return fmt.Errorf("Server is not ready to be initalized. State: %s",
			srv.state)
	}

	srv.initializeReq = req
	srv.initializeReqTime = time.Now()
	srv.state = stateInitializedUnconfirmed

	return nil
}

func (srv *server) isInitializationConfirmed() bool {
	return srv.state == stateInitializedConfirmed
}

func (srv *server) CheckInitializationIsConfirmed() error {
	if !srv.isInitializationConfirmed() {
		return srvNotInitializedErr(srv.State())
	}
	return nil
}

func (srv *server) ConfirmInitialization(req *jrpc2.Request) error {
	if srv.state != stateInitializedUnconfirmed {
		if srv.isInitializationConfirmed() {
			return fmt.Errorf("Server was already confirmed as initalized at %s via request %s",
				srv.initializedReqTime, srv.initializedReq.ID())
		}
		return fmt.Errorf("Server is not ready to be confirmed as initialized (%s).",
			srv.state)
	}
	srv.initializedReq = req
	srv.initializedReqTime = time.Now()
	srv.state = stateInitializedConfirmed

	return nil
}

func (srv *server) Shutdown(req *jrpc2.Request) error {
	if srv.isDown() {
		return srvAlreadyDownErr(srv.downReq.ID())
	}

	srv.downReq = req
	srv.downReqTime = time.Now()
	srv.state = stateDown

	return nil
}

func (srv *server) Exit() error {
	if !srv.isExitable() {
		return fmt.Errorf("Cannot exit as server is %s", srv.State())
	}
	srv.exitFunc()

	return nil
}

func (srv *server) isExitable() bool {
	return srv.isDown() || srv.isPrepared()
}

func (srv *server) isDown() bool {
	return srv.state == stateDown
}

func (srv *server) State() serverState {
	return srv.state
}

func NewServerController(exitFunc context.CancelFunc) *server {
	return &server{
		state:    stateEmpty,
		exitFunc: exitFunc,
	}
}
