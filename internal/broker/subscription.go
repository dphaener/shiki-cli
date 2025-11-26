package broker

// Subscription represents a typed event subscription.
type Subscription struct {
	id        string
	filters   []EventType
	eventCh   chan Event
	closeCh   chan struct{}
	closed    bool
}

// newSubscription creates a new subscription.
func newSubscription(id string, bufferSize int, filters []EventType) *Subscription {
	return &Subscription{
		id:      id,
		filters: filters,
		eventCh: make(chan Event, bufferSize),
		closeCh: make(chan struct{}),
	}
}

// Events returns the channel for receiving events.
func (s *Subscription) Events() <-chan Event {
	return s.eventCh
}

// Close signals that the subscription should be closed.
func (s *Subscription) Close() {
	if !s.closed {
		close(s.closeCh)
		s.closed = true
	}
}

// ID returns the subscription identifier.
func (s *Subscription) ID() string {
	return s.id
}

// wants returns true if this subscription wants the given event type.
func (s *Subscription) wants(eventType EventType) bool {
	if len(s.filters) == 0 {
		return true // No filters means subscribe to all
	}
	for _, f := range s.filters {
		if f == eventType {
			return true
		}
	}
	return false
}

// send attempts to send an event to the subscription.
// Returns false if the subscription is closed or buffer is full.
func (s *Subscription) send(event Event) bool {
	if s.closed {
		return false
	}
	select {
	case s.eventCh <- event:
		return true
	case <-s.closeCh:
		return false
	default:
		// Buffer full, drop event
		return false
	}
}
