package scheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolConcurrencyLimit(t *testing.T) {
	pool := NewPool(2)
	defer pool.Stop()

	var maxConcurrent int32
	var current int32
	var mu sync.Mutex
	done := make(chan struct{})
	var completed int32

	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			c := atomic.AddInt32(&current, 1)
			mu.Lock()
			if c > maxConcurrent {
				maxConcurrent = c
			}
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&current, -1)
			if atomic.AddInt32(&completed, 1) == 5 {
				close(done)
			}
		})
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}

	mu.Lock()
	defer mu.Unlock()
	if maxConcurrent > 2 {
		t.Errorf("max concurrent was %d, expected <= 2", maxConcurrent)
	}
	if atomic.LoadInt32(&completed) != 5 {
		t.Errorf("expected 5 completed, got %d", completed)
	}
}

func TestPoolStop(t *testing.T) {
	pool := NewPool(1)
	var ran atomic.Bool
	pool.Stop()
	pool.Submit(func() {
		ran.Store(true)
	})
	time.Sleep(50 * time.Millisecond)
	if ran.Load() {
		t.Error("job should not run after Stop")
	}
}
