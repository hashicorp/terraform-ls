// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package session

// sessionState represents state of the language server
// workspace ("session") with respect to the LSP
type sessionState int

const (
	// Before session starts
	stateEmpty sessionState = -1
	// After session starts, before any request
	statePrepared sessionState = 0
	// After "initialize", before "initialized"
	stateInitializedUnconfirmed sessionState = 1
	// After "initialized"
	stateInitializedConfirmed sessionState = 2
	// After "shutdown"
	stateDown sessionState = 3
)

func (ss sessionState) String() string {
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
