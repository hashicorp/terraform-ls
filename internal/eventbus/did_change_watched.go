// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package eventbus

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/protocol"
)

// DidChangeWatchedEvent is the event that is emitted when a client notifies
// the language server that a directory or file was changed outside of the
// editor.
type DidChangeWatchedEvent struct {
	Context context.Context

	// RawPath contains an OS specific path to the file or directory that was
	// changed. Usually extracted from the URI.
	RawPath string
	// IsDir is true if we were able to determine that the path is a directory.
	IsDir      bool
	ChangeType protocol.FileChangeType
}

func (n *EventBus) OnDidChangeWatched(identifier string, doneChannel <-chan struct{}) <-chan DidChangeWatchedEvent {
	n.logger.Printf("bus: %q subscribed to OnDidChangeWatched", identifier)
	return n.didChangeWatchedTopic.Subscribe(doneChannel)
}

func (n *EventBus) DidChangeWatched(e DidChangeWatchedEvent) {
	n.logger.Printf("bus: -> DidChangeWatched %s", e.RawPath)
	n.didChangeWatchedTopic.Publish(e)
}
