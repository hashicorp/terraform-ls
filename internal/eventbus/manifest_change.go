// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package eventbus

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/protocol"
)

// ManifestChangeEvent is an event that should be fired whenever the module
// manifest file changes.
type ManifestChangeEvent struct {
	Context context.Context

	Dir        document.DirHandle
	ChangeType protocol.FileChangeType
}

func (n *EventBus) OnManifestChange(identifier string, doneChannel <-chan struct{}) <-chan ManifestChangeEvent {
	n.logger.Printf("bus: %q subscribed to OnManifestChange", identifier)
	return n.manifestChangeTopic.Subscribe(doneChannel)
}

func (n *EventBus) ManifestChange(e ManifestChangeEvent) {
	n.logger.Printf("bus: -> ManifestChange %s", e.Dir)
	n.manifestChangeTopic.Publish(e)
}
