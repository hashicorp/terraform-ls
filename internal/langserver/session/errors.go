// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package session

import (
	"fmt"

	"github.com/creachadair/jrpc2"
)

const SessionNotInitialized jrpc2.Code = -32002

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
		return fmt.Errorf("%w: %s", jrpc2.InvalidRequest.Err(), uss)
	}

	return uss
}

func SessionAlreadyInitializedErr(reqID string) error {
	return fmt.Errorf("%w: session was already initialized via request ID %s",
		jrpc2.SystemError.Err(), reqID)
}

func SessionAlreadyDownErr(reqID string) error {
	return fmt.Errorf("%w: session was already shut down via request %s",
		jrpc2.InvalidRequest.Err(), reqID)
}

type InvalidURIErr struct {
	URI string
}

func (e *InvalidURIErr) Error() string {
	return fmt.Sprintf("invalid URI: %s", e.URI)
}
