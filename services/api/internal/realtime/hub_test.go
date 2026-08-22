package realtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
)

const wait = 2 * time.Second

// running returns a hub already pumping a bus, torn down with the test.
func running(t *testing.T) (*realtime.Hub, *bus.Memory) {
	t.Helper()
	b := bus.NewMemory()
	h := realtime.New()
	ctx, cancel := context.WithCancel(t.Context())

	// Attach subscribes before it returns, so nothing published below can
	// race the subscription.
	pump, err := h.Attach(ctx, b)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	done := make(chan struct{})
	go func() { defer close(done); _ = pump() }()
	t.Cleanup(func() {
		cancel()
		_ = b.Close()
		<-done
	})
	return h, b
}

func publish(t *testing.T, b *bus.Memory, room bus.Room, kind string) {
	t.Helper()
	if err := b.Publish(t.Context(), bus.Message{Rooms: []bus.Room{room}, Kind: kind}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func next(t *testing.T, c *realtime.Conn) realtime.Delivery {
	t.Helper()
	select {
	case d, open := <-c.Out():
		if !open {
			t.Fatal("connection queue closed while waiting")
		}
		return d
	case <-time.After(wait):
		t.Fatal("timed out waiting for a delivery")
		return realtime.Delivery{}
	}
}

func TestDeliversToAJoinedRoom(t *testing.T) {
	h, b := running(t)
	c := h.Connect()
	c.Join("project:p1")

	publish(t, b, "project:p1", "item.created")
	if got := next(t, c); got.Kind != "item.created" || got.Room != "project:p1" {
		t.Errorf("Delivery = %+v, want item.created in project:p1", got)
	}
}

// A room a client did not join must not reach it. This is the whole point of
// "everything they need, not more".
func TestDeliversNothingToAnUnjoinedRoom(t *testing.T) {
	h, b := running(t)
	watching := h.Connect()
	watching.Join("project:p1")
	elsewhere := h.Connect()
	elsewhere.Join("project:p2")

	publish(t, b, "project:p1", "item.created")
	if got := next(t, watching); got.Kind != "item.created" {
		t.Fatalf("the joined connection missed its own room: %+v", got)
	}
	select {
	case d := <-elsewhere.Out():
		t.Errorf("a connection in another room received %+v", d)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestLeaveStopsDelivery(t *testing.T) {
	h, b := running(t)
	c := h.Connect()
	c.Join("room")
	c.Leave("room")

	publish(t, b, "room", "ignored")
	select {
	case d := <-c.Out():
		t.Errorf("received %+v after leaving the room", d)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEveryJoinedConnectionReceives(t *testing.T) {
	h, b := running(t)
	conns := []*realtime.Conn{h.Connect(), h.Connect(), h.Connect()}
	for _, c := range conns {
		c.Join("room")
	}
	publish(t, b, "room", "broadcast")
	for i, c := range conns {
		if got := next(t, c); got.Kind != "broadcast" {
			t.Errorf("connection %d got %+v", i, got)
		}
	}
}

// Two different things can make a connection miss messages, and each has its
// own path to a resync. They are tested apart because a flood exercises both
// at once, which let one of them break silently.

// The hub itself fell behind: the bus dropped messages before the hub saw
// them, leaving a gap in Seq. The hub cannot know what it missed or who it
// concerned, so every connection is told to resynchronise.
//
// The gap is scripted rather than provoked. Flooding a real bus and hoping it
// drops is a race — under load the publisher is descheduled, the hub catches
// up, and no gap ever happens — so the subscriber is faked and the sequence
// numbers simply skip.
func TestAGapInTheBusResyncsEveryConnection(t *testing.T) {
	scripted := &scriptedBus{messages: make(chan bus.Message, 4)}
	h := realtime.New()
	ctx, cancel := context.WithCancel(t.Context())
	pump, err := h.Attach(ctx, scripted)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	done := make(chan struct{})
	go func() { defer close(done); _ = pump() }()
	t.Cleanup(func() {
		cancel()
		close(scripted.messages)
		<-done
	})

	c := h.Connect()
	c.Join("room")

	scripted.messages <- bus.Message{Seq: 1, Rooms: []bus.Room{"room"}, Kind: "first"}
	scripted.messages <- bus.Message{Seq: 5, Rooms: []bus.Room{"room"}, Kind: "after a gap"}

	if !awaitResync(t, c) {
		t.Error("a gap in the sequence never produced a resync")
	}
}

// scriptedBus lets a test hand the hub exactly the sequence it should see.
type scriptedBus struct{ messages chan bus.Message }

func (s *scriptedBus) Subscribe(context.Context) (<-chan bus.Message, error) {
	return s.messages, nil
}

// awaitResync reads until a resync arrives or time runs out. Only safe where
// reading cannot itself prevent what is being tested.
func awaitResync(t *testing.T, c *realtime.Conn) bool {
	t.Helper()
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		select {
		case d, open := <-c.Out():
			if !open {
				t.Fatal("the connection closed before any resync arrived")
			}
			if d.Resync {
				return true
			}
		case <-time.After(20 * time.Millisecond):
		}
	}
	return false
}

// The connection itself fell behind: the hub kept up, but this client stopped
// reading and its queue filled. Publishing stays under what the bus buffers,
// so there is no gap and only the queue can be the cause.
func TestAFullConnectionQueueResyncs(t *testing.T) {
	h, b := running(t)
	c := h.Connect()
	c.Join("room")

	// Under the bus's per-subscriber buffer so nothing is dropped there,
	// and well over the connection's own queue so that one overflows.
	for range 200 {
		publish(t, b, "room", "steady")
	}
	// Let the queue fill before reading a single message. Reading while the
	// hub is still delivering drains it as fast as it fills, so it never
	// reaches capacity and the very thing being tested never happens.
	settle(t, c)
	if !drainForResync(c) {
		t.Error("a connection whose queue filled was never told to resync")
	}
}

// settle waits until the connection's queue stops changing, so a test can be
// sure the hub has finished delivering without consuming anything.
func settle(t *testing.T, c *realtime.Conn) {
	t.Helper()
	deadline := time.Now().Add(wait)
	steady, last := 0, -1
	for time.Now().Before(deadline) {
		if n := len(c.Out()); n == last {
			if steady++; steady >= 3 {
				return
			}
		} else {
			steady, last = 0, n
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// drainForResync empties the queue, reporting whether a resync was in it.
func drainForResync(c *realtime.Conn) bool {
	found := false
	for {
		select {
		case d, open := <-c.Out():
			if !open {
				return found
			}
			found = found || d.Resync
		default:
			return found
		}
	}
}

func TestCloseRemovesTheConnection(t *testing.T) {
	h, _ := running(t)
	c := h.Connect()
	if h.Connections() != 1 {
		t.Fatalf("Connections = %d, want 1", h.Connections())
	}
	c.Close()
	if h.Connections() != 0 {
		t.Errorf("Connections = %d after Close, want 0", h.Connections())
	}
	if _, open := <-c.Out(); open {
		t.Error("the queue should be closed after Close")
	}
	c.Close() // must not panic
}

func TestJoinAndLeaveAreIdempotent(t *testing.T) {
	h, _ := running(t)
	c := h.Connect()
	c.Join("room")
	c.Join("room")
	if rooms := c.Rooms(); len(rooms) != 1 {
		t.Errorf("Rooms = %v, want one entry", rooms)
	}
	c.Leave("room")
	c.Leave("never-joined")
	if rooms := c.Rooms(); len(rooms) != 0 {
		t.Errorf("Rooms = %v, want none", rooms)
	}
}
