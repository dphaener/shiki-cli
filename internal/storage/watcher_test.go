package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
)

func TestWatcherBasicFileChange(t *testing.T) {
	wsDir := t.TempDir()
	eventChan := make(chan events.Event, 100)

	watcher, err := NewWatcher(wsDir, eventChan)
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx, "test-session")

	// Give watcher time to initialize
	time.Sleep(50 * time.Millisecond)

	// Write a file
	testFile := filepath.Join(wsDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o600)
	require.NoError(t, err)

	// Wait for event
	select {
	case event := <-eventChan:
		assert.Equal(t, types.EventFileUpdated, event.Type)
		assert.Equal(t, "test-session", event.SessionID)
		payload := event.Payload.(events.FileUpdatedPayload)
		assert.Contains(t, payload.Path, "test.txt")
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for file change event")
	}
}

func TestWatcherIgnoresTempFiles(t *testing.T) {
	wsDir := t.TempDir()
	eventChan := make(chan events.Event, 100)

	watcher, err := NewWatcher(wsDir, eventChan)
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx, "test-session")

	time.Sleep(50 * time.Millisecond)

	// Write a temp file
	tempFile := filepath.Join(wsDir, ".tmp-12345")
	err = os.WriteFile(tempFile, []byte("temp"), 0o600)
	require.NoError(t, err)

	// Should not receive event for temp file
	select {
	case event := <-eventChan:
		t.Fatalf("received unexpected event for temp file: %+v", event)
	case <-time.After(200 * time.Millisecond):
		// Expected - no event
	}
}

func TestWatcherSubdirectories(t *testing.T) {
	wsDir := t.TempDir()
	messagesDir := filepath.Join(wsDir, "messages")
	err := os.MkdirAll(messagesDir, 0o700)
	require.NoError(t, err)

	eventChan := make(chan events.Event, 100)

	watcher, err := NewWatcher(wsDir, eventChan)
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx, "test-session")

	time.Sleep(50 * time.Millisecond)

	// Write file in subdirectory
	testFile := filepath.Join(messagesDir, "message.md")
	err = os.WriteFile(testFile, []byte("content"), 0o600)
	require.NoError(t, err)

	// Wait for event
	select {
	case event := <-eventChan:
		assert.Equal(t, types.EventFileUpdated, event.Type)
		payload := event.Payload.(events.FileUpdatedPayload)
		assert.Contains(t, payload.Path, "message.md")
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for subdirectory file change event")
	}
}

func TestWatcherClose(t *testing.T) {
	wsDir := t.TempDir()
	eventChan := make(chan events.Event, 100)

	watcher, err := NewWatcher(wsDir, eventChan)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	watcher.Start(ctx, "test-session")

	// Close watcher
	cancel()
	err = watcher.Close()
	require.NoError(t, err)
}

func TestWatcherLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping latency test in short mode")
	}

	wsDir := t.TempDir()
	eventChan := make(chan events.Event, 100)

	watcher, err := NewWatcher(wsDir, eventChan)
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx, "test-session")

	time.Sleep(50 * time.Millisecond)

	// Measure latency
	iterations := 10
	var totalLatency time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		testFile := filepath.Join(wsDir, "latency-test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0o600)
		require.NoError(t, err)

		// Wait for event
		select {
		case <-eventChan:
			latency := time.Since(start)
			totalLatency += latency
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}

		// Remove file for next iteration
		_ = os.Remove(testFile)
		time.Sleep(50 * time.Millisecond)
	}

	avgLatency := totalLatency / time.Duration(iterations)
	t.Logf("Average latency: %v", avgLatency)

	// Target is <10ms, but allow some slack for test environment
	assert.Less(t, avgLatency, 100*time.Millisecond,
		"average latency %v exceeds 100ms threshold", avgLatency)
}

func BenchmarkWatcherLatency(b *testing.B) {
	wsDir := b.TempDir()
	eventChan := make(chan events.Event, 1000)

	watcher, _ := NewWatcher(wsDir, eventChan)
	defer func() { _ = watcher.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx, "bench")

	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Write file
		path := filepath.Join(wsDir, "bench.txt")
		_ = os.WriteFile(path, []byte("test"), 0o600)

		// Wait for event
		<-eventChan

		latency := time.Since(start)
		if latency > 10*time.Millisecond {
			b.Logf("High latency detected: %v", latency)
		}

		// Cleanup
		_ = os.Remove(path)
		time.Sleep(10 * time.Millisecond)
	}
}
