package whatsapp

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// HubEvent is a realtime event broadcast to a user's inbox WebSocket clients.
type HubEvent struct {
	Type    string      `json:"type"`              // "message" | "receipt" | "chat_update"
	Payload interface{} `json:"payload,omitempty"` // event-specific body
}

// hubConn wraps a websocket connection with a write mutex so concurrent
// broadcasts and ping writes don't interleave on the same socket.
type hubConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *hubConn) writeJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

// Hub fans out WhatsApp realtime events to connected clients, keyed by user ID.
type Hub struct {
	mu    sync.RWMutex
	conns map[uint]map[*hubConn]struct{}
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{conns: make(map[uint]map[*hubConn]struct{})}
}

// Register adds a connection for a user and returns the wrapped connection,
// which must be passed to Unregister when the socket closes.
func (h *Hub) Register(userID uint, conn *websocket.Conn) *hubConn {
	hc := &hubConn{conn: conn}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*hubConn]struct{})
	}
	h.conns[userID][hc] = struct{}{}
	return hc
}

// Unregister removes a connection for a user.
func (h *Hub) Unregister(userID uint, hc *hubConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.conns[userID]; ok {
		delete(set, hc)
		if len(set) == 0 {
			delete(h.conns, userID)
		}
	}
}

// WriteJSON writes a raw payload to a single connection (used for keepalive pings).
func (h *Hub) WriteJSON(hc *hubConn, v interface{}) error {
	return hc.writeJSON(v)
}

// Conn exposes the underlying websocket connection for read-side handling.
func (h *Hub) Conn(hc *hubConn) *websocket.Conn {
	return hc.conn
}

// LockWrite/UnlockWrite expose the per-conn write mutex for ping writers.
func (h *Hub) LockWrite(hc *hubConn)   { hc.mu.Lock() }
func (h *Hub) UnlockWrite(hc *hubConn) { hc.mu.Unlock() }

// Broadcast sends an event to all of a user's connected clients. Dead
// connections are pruned on write failure.
func (h *Hub) Broadcast(userID uint, evt HubEvent) {
	h.mu.RLock()
	set := h.conns[userID]
	conns := make([]*hubConn, 0, len(set))
	for hc := range set {
		conns = append(conns, hc)
	}
	h.mu.RUnlock()

	var dead []*hubConn
	for _, hc := range conns {
		if err := hc.writeJSON(evt); err != nil {
			log.Printf("[wa-hub] write error for user %d: %v", userID, err)
			dead = append(dead, hc)
		}
	}
	for _, hc := range dead {
		h.Unregister(userID, hc)
		hc.conn.Close()
	}
}
