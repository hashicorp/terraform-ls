// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package eventbus

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/protocol"
)

// PluginLockChangeEvent is an event that should be fired whenever the lock
// file changes.
type PluginLockChangeEvent struct {
	Context context.Context

	Dir        document.DirHandle
	ChangeType protocol.FileChangeType
}

func (n *EventBus) OnPluginLockChange(identifier string, doneChannel DoneChannel) <-chan PluginLockChangeEvent {
	n.logger.Printf("bus: %q subscribed to OnPluginLockChange", identifier)
	return n.pluginLockChangeTopic.Subscribe(doneChannel)
}

func (n *EventBus) PluginLockChange(e PluginLockChangeEvent) {
	n.logger.Printf("bus: -> PluginLockChange %s", e.Dir)
	n.pluginLockChangeTopic.Publish(e)
}
