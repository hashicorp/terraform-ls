// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package session

import (
	"context"
	"fmt"
	"time"

	"github.com/creachadair/jrpc2"
)

type session struct {
	initializeReq     *jrpc2.Request
	initializeReqTime time.Time

	initializedReq     *jrpc2.Request
	initializedReqTime time.Time

	downReq     *jrpc2.Request
	downReqTime time.Time

	state    sessionState
	exitFunc context.CancelFunc
}

func (s *session) isPrepared() bool {
	return s.state == statePrepared
}

func (s *session) Prepare() error {
	if s.state != stateEmpty {
		return &unexpectedSessionState{
			ExpectedState: stateInitializedConfirmed,
			CurrentState:  s.state,
		}
	}

	s.state = statePrepared

	return nil
}

func (s *session) IsInitializedUnconfirmed() bool {
	return s.state == stateInitializedUnconfirmed
}

func (s *session) Initialize(req *jrpc2.Request) error {
	if s.state != statePrepared {
		if s.IsInitializedUnconfirmed() {
			return SessionAlreadyInitializedErr(s.initializeReq.ID())
		}
		return fmt.Errorf("session is not ready to be initialized. State: %s",
			s.state)
	}

	s.initializeReq = req
	s.initializeReqTime = time.Now()
	s.state = stateInitializedUnconfirmed

	return nil
}

func (s *session) isInitializationConfirmed() bool {
	return s.state == stateInitializedConfirmed
}

func (s *session) CheckInitializationIsConfirmed() error {
	if !s.isInitializationConfirmed() {
		return SessionNotInitializedErr(s.State())
	}
	return nil
}

func (s *session) ConfirmInitialization(req *jrpc2.Request) error {
	if s.state != stateInitializedUnconfirmed {
		if s.isInitializationConfirmed() {
			return fmt.Errorf("session was already confirmed as initalized at %s via request %s",
				s.initializedReqTime, s.initializedReq.ID())
		}
		return fmt.Errorf("session is not ready to be confirmed as initialized (%s)",
			s.state)
	}
	s.initializedReq = req
	s.initializedReqTime = time.Now()
	s.state = stateInitializedConfirmed

	return nil
}

func (s *session) Shutdown(req *jrpc2.Request) error {
	if s.isDown() {
		return SessionAlreadyDownErr(s.downReq.ID())
	}

	s.downReq = req
	s.downReqTime = time.Now()
	s.state = stateDown

	return nil
}

func (s *session) Exit() error {
	if !s.isExitable() {
		return fmt.Errorf("Cannot exit as session is %s", s.State())
	}
	s.exitFunc()

	return nil
}

func (s *session) isExitable() bool {
	return s.isDown() || s.isPrepared()
}

func (s *session) isDown() bool {
	return s.state == stateDown
}

func (s *session) State() sessionState {
	return s.state
}

func NewSession(exitFunc context.CancelFunc) *session {
	return &session{
		state:    stateEmpty,
		exitFunc: exitFunc,
	}
}
