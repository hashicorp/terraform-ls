package langserver

import (
	"fmt"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/code"
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

type unexpectedSrvState struct {
	expectedState serverState
	currentState  serverState
}

func (e *unexpectedSrvState) Error() string {
	return fmt.Sprintf("server is not %s, current state: %s",
		e.expectedState, e.currentState)
}

const serverNotInitialized code.Code = -32002

func SrvNotInitializedErr(state serverState) error {
	uss := &unexpectedSrvState{
		expectedState: stateInitializedConfirmed,
		currentState:  state,
	}
	if state < stateInitializedConfirmed {
		return fmt.Errorf("%w: %s", serverNotInitialized.Err(), uss)
	}
	if state == stateDown {
		return fmt.Errorf("%w: %s", code.InvalidRequest.Err(), uss)
	}

	return uss
}

type server struct {
	initializeReq     *jrpc2.Request
	initializeReqTime time.Time

	initializedReq     *jrpc2.Request
	initializedReqTime time.Time

	state serverState

	downReq     *jrpc2.Request
	downReqTime time.Time
}

func (srv *server) IsPrepared() bool {
	return srv.state == statePrepared
}

func (srv *server) Prepare() error {
	if srv.state != stateEmpty {
		return &unexpectedSrvState{
			expectedState: stateInitializedConfirmed,
			currentState:  srv.state,
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
			return fmt.Errorf("Server was already initialized at %s via request %s",
				srv.initializeReqTime, srv.initializeReq.ID())
		}
		return fmt.Errorf("Server is not ready to be initalized. State: %s",
			srv.state)
	}

	srv.initializeReq = req
	srv.initializeReqTime = time.Now()
	srv.state = stateInitializedUnconfirmed

	return nil
}

func (srv *server) IsInitializationConfirmed() bool {
	return srv.state == stateInitializedConfirmed
}

func (srv *server) ConfirmInitialization(req *jrpc2.Request) error {
	if srv.state != stateInitializedUnconfirmed {
		if srv.IsInitializationConfirmed() {
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
	if srv.IsDown() {
		return fmt.Errorf("%w: server was already shut down via request %s",
			code.InvalidRequest.Err(), srv.downReq.ID())
	}

	srv.downReq = req
	srv.downReqTime = time.Now()
	srv.state = stateDown

	return nil
}

func (srv *server) IsDown() bool {
	return srv.state == stateDown
}

func (srv *server) State() serverState {
	return srv.state
}

func newServer() *server {
	return &server{state: stateEmpty}
}
