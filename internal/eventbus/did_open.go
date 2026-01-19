// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package eventbus

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
)

// DidOpenEvent is an event to signal that a directory is open in the editor
// or important for the language server to process.
//
// It is usually emitted when a document is opened via a language server
// text synchronization event.
// It can also be fired from different features to signal that a directory
// should be processed in other features as well.
type DidOpenEvent struct {
	Context context.Context

	Dir        document.DirHandle
	LanguageID string
}

func (n *EventBus) OnDidOpen(identifier string, doneChannel DoneChannel) <-chan DidOpenEvent {
	n.logger.Printf("bus: %q subscribed to OnDidOpen", identifier)
	return n.didOpenTopic.Subscribe(doneChannel)
}

func (n *EventBus) DidOpen(e DidOpenEvent) job.IDs {
	n.logger.Printf("bus: -> DidOpen %s", e.Dir)
	return n.didOpenTopic.Publish(e)
}
