package events

import (
	"testing"
	"time"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBusPublishSubscribe(t *testing.T) {
	bus := NewEventBus(10)
	defer bus.Shutdown()

	sub := bus.Subscribe("test")

	event := Event{
		Type:      types.EventTurnStarted,
		SessionID: "test-session",
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	select {
	case received := <-sub.Events():
		assert.Equal(t, event.Type, received.Type)
		assert.Equal(t, event.SessionID, received.SessionID)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestSubscriberFiltering(t *testing.T) {
	bus := NewEventBus(10)
	defer bus.Shutdown()

	// Subscribe to specific event types
	sub1 := bus.Subscribe("sub1", types.EventTurnStarted, types.EventTurnCompleted)
	sub2 := bus.Subscribe("sub2", types.EventMessageSent)

	// Publish various events
	bus.Publish(Event{Type: types.EventTurnStarted, SessionID: "test"})
	bus.Publish(Event{Type: types.EventMessageSent, SessionID: "test"})
	bus.Publish(Event{Type: types.EventSessionCreated, SessionID: "test"})

	time.Sleep(50 * time.Millisecond) // Let events distribute

	// Verify sub1 received only TurnStarted
	select {
	case event := <-sub1.Events():
		assert.Equal(t, types.EventTurnStarted, event.Type)
	case <-time.After(time.Second):
		t.Fatal("sub1 timeout")
	}

	// Verify no more events for sub1
	select {
	case <-sub1.Events():
		t.Fatal("sub1 should not receive SessionCreated")
	case <-time.After(100 * time.Millisecond):
		// Expected: no event
	}

	// Verify sub2 received only MessageSent
	select {
	case event := <-sub2.Events():
		assert.Equal(t, types.EventMessageSent, event.Type)
	case <-time.After(time.Second):
		t.Fatal("sub2 timeout")
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewEventBus(10)
	defer bus.Shutdown()

	sub := bus.Subscribe("test")
	bus.Publish(Event{Type: types.EventTurnStarted, SessionID: "test", Timestamp: time.Now()})

	// Receive first event
	select {
	case <-sub.Events():
		// Expected
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Unsubscribe
	bus.Unsubscribe("test")

	// Channel should be closed
	_, ok := <-sub.Events()
	assert.False(t, ok, "channel should be closed after unsubscribe")
}

func TestSubscribeToAllEvents(t *testing.T) {
	bus := NewEventBus(10)
	defer bus.Shutdown()

	// Subscribe without specifying event types (receive all)
	sub := bus.Subscribe("all-events")

	events := []Event{
		{Type: types.EventTurnStarted, SessionID: "test", Timestamp: time.Now()},
		{Type: types.EventMessageSent, SessionID: "test", Timestamp: time.Now()},
		{Type: types.EventSessionCreated, SessionID: "test", Timestamp: time.Now()},
	}

	for _, event := range events {
		bus.Publish(event)
	}

	time.Sleep(50 * time.Millisecond)

	// Should receive all 3 events
	for i := 0; i < 3; i++ {
		select {
		case <-sub.Events():
			// Expected
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for event %d", i)
		}
	}
}

func TestGracefulShutdown(t *testing.T) {
	bus := NewEventBus(100)

	sub1 := bus.Subscribe("sub1")
	sub2 := bus.Subscribe("sub2")

	// Publish some events
	for i := 0; i < 10; i++ {
		bus.Publish(Event{Type: types.EventTurnStarted, SessionID: "test", Timestamp: time.Now()})
	}

	// Shutdown should complete quickly
	start := time.Now()
	err := bus.Shutdown()
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, elapsed, time.Second, "shutdown should complete in <1s")

	// Drain any remaining events and verify channels are closed
	drainChannel := func(ch <-chan Event) bool {
		for {
			_, ok := <-ch
			if !ok {
				return true // Channel closed
			}
		}
	}

	assert.True(t, drainChannel(sub1.Events()), "sub1 channel should be closed")
	assert.True(t, drainChannel(sub2.Events()), "sub2 channel should be closed")
}

func TestNonBlockingPublish(t *testing.T) {
	bus := NewEventBus(10)
	defer bus.Shutdown()

	// Create slow subscriber
	sub := bus.Subscribe("slow")
	go func() {
		for range sub.Events() {
			time.Sleep(100 * time.Millisecond) // Very slow consumer
		}
	}()

	// Rapidly publish many events
	start := time.Now()
	for i := 0; i < 100; i++ {
		bus.Publish(Event{Type: types.EventTurnStarted, SessionID: "test", Timestamp: time.Now()})
	}
	elapsed := time.Since(start)

	// Publishing should be fast despite slow subscriber
	assert.Less(t, elapsed, 100*time.Millisecond, "publish should not block on slow subscriber")
}

func TestStats(t *testing.T) {
	bus := NewEventBus(50)
	defer bus.Shutdown()

	bus.Subscribe("sub1")
	bus.Subscribe("sub2")
	bus.Subscribe("sub3")

	stats := bus.Stats()

	assert.Equal(t, 3, stats["subscribers"])
	assert.Equal(t, 50, stats["buffer_size"])
	assert.Equal(t, 0, stats["buffered_events"])
}

// Benchmark publish latency (target: <10μs for non-blocking operation)
func BenchmarkPublish(b *testing.B) {
	bus := NewEventBus(1000) // Large buffer to ensure non-blocking
	defer bus.Shutdown()

	event := Event{
		Type:      types.EventTurnStarted,
		SessionID: "test",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(event)
	}
	b.StopTimer()

	// Verify <10μs per publish (non-blocking operation)
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	b.Logf("Publish latency: %dns", nsPerOp)
	if nsPerOp > 10000 {
		b.Fatalf("publish too slow: %dns (want <10000ns)", nsPerOp)
	}
}

// Benchmark throughput (target: >10k events/sec)
func BenchmarkThroughput(b *testing.B) {
	bus := NewEventBus(1000)
	defer bus.Shutdown()

	// Create slow subscriber
	sub := bus.Subscribe("slow", types.EventTurnStarted)
	go func() {
		for range sub.Events() {
			time.Sleep(10 * time.Millisecond) // Slow consumer
		}
	}()

	event := Event{
		Type:      types.EventTurnStarted,
		SessionID: "test",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	start := time.Now()
	for i := 0; i < 10000; i++ {
		bus.Publish(event)
	}
	elapsed := time.Since(start)
	b.StopTimer()

	// Verify >10k events/sec
	eventsPerSec := float64(10000) / elapsed.Seconds()
	if eventsPerSec < 10000 {
		b.Fatalf("throughput too low: %.0f events/sec (want >10000)", eventsPerSec)
	}
	b.Logf("Throughput: %.0f events/sec", eventsPerSec)
}
