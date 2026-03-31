package eventbus

import (
	"sync"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	var received map[string]any
	var mu sync.Mutex
	done := make(chan struct{})

	bus.Subscribe("test_event", func(payload map[string]any) {
		mu.Lock()
		received = payload
		mu.Unlock()
		close(done)
	})

	bus.Publish("test_event", map[string]any{"key": "value"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	mu.Lock()
	defer mu.Unlock()
	if received["key"] != "value" {
		t.Errorf("got %v, want key=value", received)
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	called := false
	unsub := bus.Subscribe("test_event", func(payload map[string]any) {
		called = true
	})

	unsub()
	bus.Publish("test_event", map[string]any{})
	time.Sleep(50 * time.Millisecond)

	if called {
		t.Error("handler should not have been called after unsubscribe")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()

	var count int
	var mu sync.Mutex
	done := make(chan struct{})

	for i := 0; i < 3; i++ {
		bus.Subscribe("test_event", func(payload map[string]any) {
			mu.Lock()
			count++
			if count == 3 {
				close(done)
			}
			mu.Unlock()
		})
	}

	bus.Publish("test_event", map[string]any{})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for all subscribers")
	}

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("expected 3 calls, got %d", count)
	}
}

func TestNoSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()
	bus.Publish("nonexistent", map[string]any{})
}
