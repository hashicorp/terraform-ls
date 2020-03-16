package svcctl

import (
	"context"
	"fmt"
	"time"

	"github.com/creachadair/jrpc2"
)

// serviceState represents state of the language server
// workspace ("service") with respect to the LSP
type serviceState int

const (
	// Before service starts
	stateEmpty serviceState = -1
	// After service starts, before any request
	statePrepared serviceState = 0
	// After "initialize", before "initialized"
	stateInitializedUnconfirmed serviceState = 1
	// After "initialized"
	stateInitializedConfirmed serviceState = 2
	// After "shutdown"
	stateDown serviceState = 3
)

func (ss serviceState) String() string {
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

type service struct {
	initializeReq     *jrpc2.Request
	initializeReqTime time.Time

	initializedReq     *jrpc2.Request
	initializedReqTime time.Time

	downReq     *jrpc2.Request
	downReqTime time.Time

	state    serviceState
	exitFunc context.CancelFunc
}

func (svc *service) isPrepared() bool {
	return svc.state == statePrepared
}

func (svc *service) Prepare() error {
	if svc.state != stateEmpty {
		return &unexpectedSvcState{
			ExpectedState: stateInitializedConfirmed,
			CurrentState:  svc.state,
		}
	}

	svc.state = statePrepared

	return nil
}

func (svc *service) IsInitializedUnconfirmed() bool {
	return svc.state == stateInitializedUnconfirmed
}

func (svc *service) Initialize(req *jrpc2.Request) error {
	if svc.state != statePrepared {
		if svc.IsInitializedUnconfirmed() {
			return svcAlreadyInitializedErr(svc.initializeReq.ID())
		}
		return fmt.Errorf("service is not ready to be initalized. State: %s",
			svc.state)
	}

	svc.initializeReq = req
	svc.initializeReqTime = time.Now()
	svc.state = stateInitializedUnconfirmed

	return nil
}

func (svc *service) isInitializationConfirmed() bool {
	return svc.state == stateInitializedConfirmed
}

func (svc *service) CheckInitializationIsConfirmed() error {
	if !svc.isInitializationConfirmed() {
		return svcNotInitializedErr(svc.State())
	}
	return nil
}

func (svc *service) ConfirmInitialization(req *jrpc2.Request) error {
	if svc.state != stateInitializedUnconfirmed {
		if svc.isInitializationConfirmed() {
			return fmt.Errorf("service was already confirmed as initalized at %s via request %s",
				svc.initializedReqTime, svc.initializedReq.ID())
		}
		return fmt.Errorf("service is not ready to be confirmed as initialized (%s).",
			svc.state)
	}
	svc.initializedReq = req
	svc.initializedReqTime = time.Now()
	svc.state = stateInitializedConfirmed

	return nil
}

func (svc *service) Shutdown(req *jrpc2.Request) error {
	if svc.isDown() {
		return svcAlreadyDownErr(svc.downReq.ID())
	}

	svc.downReq = req
	svc.downReqTime = time.Now()
	svc.state = stateDown

	return nil
}

func (svc *service) Exit() error {
	if !svc.isExitable() {
		return fmt.Errorf("Cannot exit as service is %s", svc.State())
	}
	svc.exitFunc()

	return nil
}

func (svc *service) isExitable() bool {
	return svc.isDown() || svc.isPrepared()
}

func (svc *service) isDown() bool {
	return svc.state == stateDown
}

func (svc *service) State() serviceState {
	return svc.state
}

func NewServiceController(exitFunc context.CancelFunc) *service {
	return &service{
		state:    stateEmpty,
		exitFunc: exitFunc,
	}
}
