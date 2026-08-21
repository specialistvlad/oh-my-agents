// Package realtime routes what happened to the connections that care.
//
// It sits between [bus.Subscriber] and a transport, and knows about neither
// WebSockets nor storage: connections arrive through [Hub.Connect] and
// messages through the bus. That is what makes the whole thing testable
// without opening a socket, which is how its own tests run.
package realtime

import (
	"context"
	"sync"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
)

// Hub fans messages out to connections by room.
type Hub struct {
	mu    sync.RWMutex
	conns map[*Conn]struct{}
	seq   uint64
	begun bool
}

// New returns a hub with no connections.
func New() *Hub {
	return &Hub{conns: make(map[*Conn]struct{})}
}

// Attach subscribes the hub to a bus and returns a function that pumps until
// the context ends.
//
// Subscribing happens before Attach returns, which is the point of splitting
// the two: `go h.Run(ctx, b)` would let a caller publish before the
// subscription existed, and the bus does not replay, so that message would be
// lost. Here anything published after Attach returns is guaranteed to be seen.
// A failure to subscribe also surfaces immediately rather than inside a
// goroutine nobody is watching.
func (h *Hub) Attach(ctx context.Context, sub bus.Subscriber) (func() error, error) {
	messages, err := sub.Subscribe(ctx)
	if err != nil {
		return nil, err
	}
	return func() error {
		for m := range messages {
			h.dispatch(m)
		}
		return ctx.Err()
	}, nil
}

// dispatch delivers one message to every connection in one of its rooms.
//
// A gap in the bus's sequence means the hub itself fell behind, so every
// connection is told to resynchronise: the hub cannot know what it missed or
// who it concerned, and guessing would be worse than saying so.
func (h *Hub) dispatch(m bus.Message) {
	h.mu.Lock()
	gap := h.begun && m.Seq > h.seq+1
	h.begun, h.seq = true, m.Seq
	conns := make([]*Conn, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()

	for _, c := range conns {
		if gap {
			c.resync()
			continue
		}
		for _, room := range m.Rooms {
			if c.joined(room) {
				c.deliver(Delivery{Room: room, Seq: m.Seq, Kind: m.Kind, Data: m.Data})
				break // one delivery per message, however many rooms match
			}
		}
	}
}

// Connect registers a connection. The caller closes it when its transport
// goes away, which also removes it from the hub.
func (h *Hub) Connect() *Conn {
	c := newConn(h)
	h.mu.Lock()
	h.conns[c] = struct{}{}
	h.mu.Unlock()
	return c
}

// Connections reports how many are registered, for tests and for a metric.
func (h *Hub) Connections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

func (h *Hub) forget(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, c)
}
