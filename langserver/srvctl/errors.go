package srvctl

import (
	"fmt"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/langserver/errors"
)

type unexpectedSrvState struct {
	ExpectedState serverState
	CurrentState  serverState
}

func (e *unexpectedSrvState) Error() string {
	return fmt.Sprintf("server is not %s, current state: %s",
		e.ExpectedState, e.CurrentState)
}

func srvNotInitializedErr(state serverState) error {
	uss := &unexpectedSrvState{
		ExpectedState: stateInitializedConfirmed,
		CurrentState:  state,
	}
	if state < stateInitializedConfirmed {
		return fmt.Errorf("%w: %s", errors.ServerNotInitialized.Err(), uss)
	}
	if state == stateDown {
		return fmt.Errorf("%w: %s", code.InvalidRequest.Err(), uss)
	}

	return uss
}

func srvAlreadyInitializedErr(reqID string) error {
	return fmt.Errorf("%w: Server was already initialized via request ID %s",
		code.SystemError.Err(), reqID)
}

func srvAlreadyDownErr(reqID string) error {
	return fmt.Errorf("%w: server was already shut down via request %s",
		code.InvalidRequest.Err(), reqID)
}
