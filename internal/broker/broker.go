package broker

import (
	"context"
	"sync"
)

const (
	// DefaultBufferSize is the default buffer size for event channels.
	DefaultBufferSize = 100

	// DefaultPublishBuffer is the default buffer for the publish channel.
	DefaultPublishBuffer = 1000
)

// Broker manages typed event distribution.
type Broker struct {
	subscribers map[string]*Subscription
	mu          sync.RWMutex
	publishCh   chan Event
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// New creates a new event broker.
func New() *Broker {
	return NewWithBuffer(DefaultPublishBuffer)
}

// NewWithBuffer creates a new event broker with a custom buffer size.
func NewWithBuffer(bufferSize int) *Broker {
	ctx, cancel := context.WithCancel(context.Background())
	b := &Broker{
		subscribers: make(map[string]*Subscription),
		publishCh:   make(chan Event, bufferSize),
		ctx:         ctx,
		cancel:      cancel,
	}
	b.wg.Add(1)
	go b.publishLoop()
	return b
}

// Subscribe registers for events with optional type filtering.
// If no filters are provided, the subscription receives all events.
func (b *Broker) Subscribe(id string, filters ...EventType) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	// If subscription already exists, close it first
	if existing, ok := b.subscribers[id]; ok {
		existing.Close()
		close(existing.eventCh)
		delete(b.subscribers, id)
	}

	sub := newSubscription(id, DefaultBufferSize, filters)
	b.subscribers[id] = sub
	return sub
}

// SubscribeWithBuffer registers for events with a custom buffer size.
func (b *Broker) SubscribeWithBuffer(id string, bufferSize int, filters ...EventType) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	// If subscription already exists, close it first
	if existing, ok := b.subscribers[id]; ok {
		existing.Close()
		close(existing.eventCh)
		delete(b.subscribers, id)
	}

	sub := newSubscription(id, bufferSize, filters)
	b.subscribers[id] = sub
	return sub
}

// Unsubscribe removes a subscription.
func (b *Broker) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, ok := b.subscribers[id]; ok {
		sub.Close()
		close(sub.eventCh)
		delete(b.subscribers, id)
	}
}

// Publish sends an event to all matching subscribers.
func (b *Broker) Publish(event Event) {
	select {
	case b.publishCh <- event:
	case <-b.ctx.Done():
	}
}

// PublishAsync publishes without blocking (drops if buffer full).
func (b *Broker) PublishAsync(event Event) bool {
	select {
	case b.publishCh <- event:
		return true
	case <-b.ctx.Done():
		return false
	default:
		// Buffer full
		return false
	}
}

// publishLoop is the main event distribution loop.
func (b *Broker) publishLoop() {
	defer b.wg.Done()

	for {
		select {
		case <-b.ctx.Done():
			return
		case event := <-b.publishCh:
			b.distribute(event)
		}
	}
}

// distribute sends an event to all matching subscribers.
func (b *Broker) distribute(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subscribers {
		if sub.wants(event.EventType()) {
			sub.send(event)
		}
	}
}

// SubscriberCount returns the number of active subscribers.
func (b *Broker) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// Shutdown gracefully shuts down the broker.
func (b *Broker) Shutdown() {
	b.cancel()
	close(b.publishCh)
	b.wg.Wait()

	b.mu.Lock()
	defer b.mu.Unlock()

	for id, sub := range b.subscribers {
		sub.Close()
		close(sub.eventCh)
		delete(b.subscribers, id)
	}
}

// SessionEventFilter returns common event types for session monitoring.
func SessionEventFilter() []EventType {
	return []EventType{
		EventSessionCreated,
		EventSessionUpdated,
		EventSessionStarted,
		EventSessionPaused,
		EventSessionCompleted,
		EventSessionError,
	}
}

// MessageEventFilter returns common event types for message monitoring.
func MessageEventFilter() []EventType {
	return []EventType{
		EventMessageCreated,
		EventMessageUpdated,
		EventPartAdded,
	}
}

// StreamEventFilter returns common event types for streaming monitoring.
func StreamEventFilter() []EventType {
	return []EventType{
		EventStreamStarted,
		EventStreamDelta,
		EventStreamFinished,
	}
}

// ToolEventFilter returns common event types for tool monitoring.
func ToolEventFilter() []EventType {
	return []EventType{
		EventToolStarted,
		EventToolCompleted,
	}
}

// TurnEventFilter returns common event types for turn monitoring.
func TurnEventFilter() []EventType {
	return []EventType{
		EventTurnStarted,
		EventTurnCompleted,
		EventTurnError,
	}
}

// TUIEventFilter returns all event types needed by the TUI.
func TUIEventFilter() []EventType {
	return []EventType{
		EventSessionCreated,
		EventSessionUpdated,
		EventSessionStarted,
		EventSessionPaused,
		EventSessionCompleted,
		EventSessionError,
		EventMessageCreated,
		EventMessageUpdated,
		EventPartAdded,
		EventStreamStarted,
		EventStreamDelta,
		EventStreamFinished,
		EventToolStarted,
		EventToolCompleted,
		EventTurnStarted,
		EventTurnCompleted,
		EventTurnError,
	}
}
