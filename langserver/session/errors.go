package session

import (
	"fmt"

	"github.com/creachadair/jrpc2/code"
)

const SessionNotInitialized code.Code = -32002

type unexpectedSessionState struct {
	ExpectedState sessionState
	CurrentState  sessionState
}

func (e *unexpectedSessionState) Error() string {
	return fmt.Sprintf("session is not %s, current state: %s",
		e.ExpectedState, e.CurrentState)
}

func SessionNotInitializedErr(state sessionState) error {
	uss := &unexpectedSessionState{
		ExpectedState: stateInitializedConfirmed,
		CurrentState:  state,
	}
	if state < stateInitializedConfirmed {
		return fmt.Errorf("%w: %s", SessionNotInitialized.Err(), uss)
	}
	if state == stateDown {
		return fmt.Errorf("%w: %s", code.InvalidRequest.Err(), uss)
	}

	return uss
}

func SessionAlreadyInitializedErr(reqID string) error {
	return fmt.Errorf("%w: session was already initialized via request ID %s",
		code.SystemError.Err(), reqID)
}

func SessionAlreadyDownErr(reqID string) error {
	return fmt.Errorf("%w: session was already shut down via request %s",
		code.InvalidRequest.Err(), reqID)
}
