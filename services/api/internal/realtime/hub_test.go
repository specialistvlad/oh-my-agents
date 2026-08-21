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

// A client too far behind is told to resynchronise rather than silently
// missing a message: behind but correct beats up to date but wrong.
func TestFloodedConnectionIsToldToResync(t *testing.T) {
	h, b := running(t)
	c := h.Connect()
	c.Join("room")

	for range 5000 {
		publish(t, b, "room", "flood")
	}
	sawResync := false
	for range 5000 {
		select {
		case d := <-c.Out():
			if d.Resync {
				sawResync = true
			}
		default:
		}
		if sawResync {
			break
		}
	}
	if !sawResync {
		t.Error("a connection that fell far behind was never told to resync")
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
