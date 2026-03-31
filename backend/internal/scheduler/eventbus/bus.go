package eventbus

import "sync"

type handler struct {
	id int
	fn func(payload map[string]any)
}

type Bus struct {
	mu     sync.RWMutex
	subs   map[string][]handler
	nextID int
	closed bool
}

func New() *Bus {
	return &Bus{subs: make(map[string][]handler)}
}

func (b *Bus) Subscribe(event string, fn func(payload map[string]any)) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextID
	b.nextID++
	b.subs[event] = append(b.subs[event], handler{id: id, fn: fn})
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		handlers := b.subs[event]
		for i, h := range handlers {
			if h.id == id {
				b.subs[event] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

func (b *Bus) Publish(event string, payload map[string]any) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	for _, h := range b.subs[event] {
		go h.fn(payload)
	}
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
}
