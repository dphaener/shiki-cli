package events

import "github.com/dphaener/shiki-cli/pkg/types"

// Subscriber represents an event consumer with filtering
type Subscriber struct {
	id         string
	eventTypes map[types.EventType]bool
	eventChan  chan Event
}

// newSubscriber creates a new subscriber with optional event type filtering
func newSubscriber(id string, eventTypes []types.EventType, bufferSize int) *Subscriber {
	typeMap := make(map[types.EventType]bool)
	for _, et := range eventTypes {
		typeMap[et] = true
	}

	return &Subscriber{
		id:         id,
		eventTypes: typeMap,
		eventChan:  make(chan Event, bufferSize),
	}
}

// wantsEvent checks if this subscriber is interested in the given event type
func (s *Subscriber) wantsEvent(eventType types.EventType) bool {
	// Empty event types map means subscribe to all events
	if len(s.eventTypes) == 0 {
		return true
	}
	return s.eventTypes[eventType]
}

// Events returns the read-only channel for receiving events
func (s *Subscriber) Events() <-chan Event {
	return s.eventChan
}
