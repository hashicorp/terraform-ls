// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package eventbus

import "context"

// DiscoverEvent is an event that is triggered by the walker when a new
// directory is walked.
//
// Most features use this to create a state entry if the directory contains
// files relevant to them.
type DiscoverEvent struct {
	Context context.Context

	Path  string
	Files []string
}

func (n *EventBus) OnDiscover(identifier string, doneChannel DoneChannel) <-chan DiscoverEvent {
	n.logger.Printf("bus: %q subscribed to OnDiscover", identifier)
	return n.discoverTopic.Subscribe(doneChannel)
}

func (n *EventBus) Discover(e DiscoverEvent) {
	n.logger.Printf("bus: -> Discover %s", e.Path)
	n.discoverTopic.Publish(e)
}
