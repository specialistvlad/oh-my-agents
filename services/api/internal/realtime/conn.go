package realtime

import (
	"encoding/json"
	"sync"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
)

// outbound is how far one connection may fall behind before it is told to
// resynchronise instead. A socket that cannot keep up with this is not going
// to be rescued by a deeper queue.
const outbound = 64

// Delivery is one message on its way to a client.
//
// Resync is the other kind of thing that can arrive: it means the connection
// missed something, and the client's answer is always to fetch the current
// state of what it is showing again. Nothing is replayed. That is the same
// move a client makes on connect, which is why there is only one recovery
// path in this design, and this is how it is signaled.
type Delivery struct {
	Room   bus.Room
	Seq    uint64
	Kind   string
	Data   json.RawMessage
	Resync bool
}

// Conn is one client's membership and outbound queue. It knows nothing about
// how it is connected.
type Conn struct {
	hub *Hub

	mu     sync.RWMutex
	rooms  map[bus.Room]struct{}
	closed bool

	out chan Delivery
}

func newConn(h *Hub) *Conn {
	return &Conn{hub: h, rooms: make(map[bus.Room]struct{}), out: make(chan Delivery, outbound)}
}

// Out is the queue a transport writes from. It is closed by [Conn.Close].
func (c *Conn) Out() <-chan Delivery { return c.out }

// Join subscribes to a room. Joining twice is not an error: the client's
// intent is already satisfied.
func (c *Conn) Join(room bus.Room) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.rooms[room] = struct{}{}
	}
}

// Leave unsubscribes. Leaving a room never joined is not an error either.
func (c *Conn) Leave(room bus.Room) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rooms, room)
}

// Rooms lists what this connection is subscribed to, in no order.
func (c *Conn) Rooms() []bus.Room {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]bus.Room, 0, len(c.rooms))
	for r := range c.rooms {
		out = append(out, r)
	}
	return out
}

// Close removes the connection from its hub and closes its queue. It is safe
// to call more than once, because a transport tearing down has more than one
// path to here.
func (c *Conn) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	c.hub.forget(c)
	close(c.out)
}

func (c *Conn) joined(room bus.Room) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.rooms[room]
	return ok
}

// deliver queues a message, or turns it into a resync if the client is too
// far behind. Dropping one message silently would leave the client wrong
// without knowing it; a resync leaves it behind but correct.
func (c *Conn) deliver(d Delivery) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return
	}
	select {
	case c.out <- d:
	default:
		c.pushResync(d.Room)
	}
}

func (c *Conn) resync() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.closed {
		c.pushResync("")
	}
}

// pushResync empties the queue and leaves a single resync in it.
//
// Dropping the resync when the queue is full would lose it exactly when it
// matters, since the queue being full is what caused it. Draining is not a
// workaround for that but the right answer on its own: the client is about to
// refetch current state, so everything queued is already superseded and
// delivering it first would be work for nothing.
func (c *Conn) pushResync(room bus.Room) {
	for {
		select {
		case <-c.out:
		default:
			select {
			case c.out <- Delivery{Room: room, Resync: true}:
			default: // a reader refilled it; it is getting a resync either way
			}
			return
		}
	}
}
