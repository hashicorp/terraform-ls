// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package eventbus

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
)

// DidChangeEvent is an event to signal that a file in directory has changed.
//
// It is usually emitted when a document is changed via a language server
// text synchronization event.
type DidChangeEvent struct {
	Context context.Context

	Dir        document.DirHandle
	LanguageID string
}

func (n *EventBus) OnDidChange(identifier string, doneChannel <-chan struct{}) <-chan DidChangeEvent {
	n.logger.Printf("bus: %q subscribed to OnDidChange", identifier)
	return n.didChangeTopic.Subscribe(doneChannel)
}

func (n *EventBus) DidChange(e DidChangeEvent) {
	n.logger.Printf("bus: -> DidChange %s", e.Dir)
	n.didChangeTopic.Publish(e)
}
