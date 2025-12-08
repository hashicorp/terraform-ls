// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package eventbus

import (
	"io"
	"log"
	"sync"

	"github.com/hashicorp/terraform-ls/internal/job"
)

const ChannelSize = 10

var discardLogger = log.New(io.Discard, "", 0)

// EventBus is a simple event bus that allows for subscribing to and publishing
// events of a specific type.
//
// It has a static list of topics. Each topic can have multiple subscribers.
// When an event is published to a topic, it is sent to all subscribers.
type EventBus struct {
	logger *log.Logger

	didOpenTopic          *Topic[DidOpenEvent]
	didChangeTopic        *Topic[DidChangeEvent]
	didChangeWatchedTopic *Topic[DidChangeWatchedEvent]
	discoverTopic         *Topic[DiscoverEvent]

	manifestChangeTopic   *Topic[ManifestChangeEvent]
	pluginLockChangeTopic *Topic[PluginLockChangeEvent]
}

func NewEventBus() *EventBus {
	return &EventBus{
		logger:                discardLogger,
		didOpenTopic:          NewTopic[DidOpenEvent](),
		didChangeTopic:        NewTopic[DidChangeEvent](),
		didChangeWatchedTopic: NewTopic[DidChangeWatchedEvent](),
		discoverTopic:         NewTopic[DiscoverEvent](),
		manifestChangeTopic:   NewTopic[ManifestChangeEvent](),
		pluginLockChangeTopic: NewTopic[PluginLockChangeEvent](),
	}
}

func (eb *EventBus) SetLogger(logger *log.Logger) {
	eb.logger = logger
}

// Topic represents a generic subscription topic
type Topic[T any] struct {
	subscribers []Subscriber[T]
	mutex       sync.Mutex
}

type DoneChannel <-chan job.IDs

// Subscriber represents a subscriber to a topic
type Subscriber[T any] struct {
	// channel is the channel to which all events of the topic are sent
	channel chan<- T

	// doneChannel is an optional channel that the subscriber can use to signal
	// that it is done processing the event
	doneChannel DoneChannel
}

// NewTopic creates a new topic
func NewTopic[T any]() *Topic[T] {
	return &Topic[T]{
		subscribers: make([]Subscriber[T], 0),
	}
}

// Subscribe adds a subscriber to a topic
func (eb *Topic[T]) Subscribe(doneChannel DoneChannel) <-chan T {
	channel := make(chan T, ChannelSize)
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	eb.subscribers = append(eb.subscribers, Subscriber[T]{
		channel:     channel,
		doneChannel: doneChannel,
	})
	return channel
}

// Publish sends an event to all subscribers of a specific topic
func (eb *Topic[T]) Publish(event T) job.IDs {
	ids := make(job.IDs, 0)

	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	for _, subscriber := range eb.subscribers {
		// Send the event to the subscriber
		subscriber.channel <- event

		if subscriber.doneChannel != nil {
			// And wait until the subscriber is done processing it
			spawnedIds := <-subscriber.doneChannel
			ids = append(ids, spawnedIds...)
		}
	}

	return ids
}
