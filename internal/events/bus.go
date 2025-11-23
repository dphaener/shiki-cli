package events

import (
	"context"
	"sync"

	"github.com/darinhaener/collab/pkg/types"
)

// EventBus manages event distribution to subscribers using channels
type EventBus struct {
	subscribers map[string]*Subscriber
	mu          sync.RWMutex
	publishChan chan Event
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewEventBus creates a new event bus with specified buffer size
func NewEventBus(bufferSize int) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus{
		subscribers: make(map[string]*Subscriber),
		publishChan: make(chan Event, bufferSize),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start publisher goroutine
	go bus.publishLoop()

	return bus
}

// Publish sends event to all subscribers (non-blocking)
func (bus *EventBus) Publish(event Event) {
	select {
	case bus.publishChan <- event:
		// Event queued successfully
	case <-bus.ctx.Done():
		// Bus shutdown, drop event
	default:
		// Buffer full, drop event (graceful degradation)
		// TODO: increment dropped event counter for observability
	}
}

// publishLoop runs in a goroutine and distributes events to subscribers
func (bus *EventBus) publishLoop() {
	for {
		select {
		case <-bus.ctx.Done():
			return
		case event := <-bus.publishChan:
			bus.distributeEvent(event)
		}
	}
}

// distributeEvent sends an event to all interested subscribers
func (bus *EventBus) distributeEvent(event Event) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	for _, sub := range bus.subscribers {
		// Check if subscriber wants this event type
		if sub.wantsEvent(event.Type) {
			select {
			case sub.eventChan <- event:
				// Event delivered successfully
			default:
				// Subscriber channel full, drop event
				// This is intentional: slow subscribers don't block fast ones
			}
		}
	}
}

// Subscribe registers a new subscriber with optional event type filtering
// If no event types are provided, subscriber receives all events
func (bus *EventBus) Subscribe(id string, eventTypes ...types.EventType) *Subscriber {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	sub := newSubscriber(id, eventTypes, 100)
	bus.subscribers[id] = sub

	return sub
}

// Unsubscribe removes subscriber and closes its channel
func (bus *EventBus) Unsubscribe(id string) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if sub, ok := bus.subscribers[id]; ok {
		close(sub.eventChan)
		delete(bus.subscribers, id)
	}
}

// Shutdown closes publisher and drains all subscribers
func (bus *EventBus) Shutdown() error {
	// Signal shutdown to publisher
	bus.cancel()

	// Close publish channel
	close(bus.publishChan)

	// Drain remaining events from publish channel
	for range bus.publishChan {
		// Discard remaining events
	}

	// Close all subscriber channels
	bus.mu.Lock()
	for _, sub := range bus.subscribers {
		close(sub.eventChan)
	}
	bus.subscribers = make(map[string]*Subscriber)
	bus.mu.Unlock()

	return nil
}

// Stats returns event bus statistics for observability
func (bus *EventBus) Stats() map[string]interface{} {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	return map[string]interface{}{
		"subscribers":     len(bus.subscribers),
		"buffer_size":     cap(bus.publishChan),
		"buffered_events": len(bus.publishChan),
	}
}
