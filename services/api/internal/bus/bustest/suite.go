// Package bustest is the conformance suite for [bus.Bus].
//
// ADR-0008 puts two implementations behind one port — an in-process one and
// Valkey — so that the default installs nothing and scaling out is a config
// change. Two implementations are only interchangeable if they behave
// identically, and this is what says they do.
package bustest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
)

// wait is how long a test will sit for a message that should already be on
// its way. Generous, because it only elapses when something is broken.
const wait = 2 * time.Second

// Factory builds a fresh, empty bus for one subtest.
type Factory func(t *testing.T) bus.Bus

// Run exercises every guarantee [bus.Bus] documents.
func Run(t *testing.T, newBus Factory) {
	t.Helper()
	for name, check := range map[string]func(*testing.T, Factory){
		"delivers to a subscriber":     testDelivers,
		"delivers to every subscriber": testFanOut,
		"preserves order":              testOrder,
		"numbers messages":             testSeq,
		"ends with its context":        testContextEnds,
		"a slow subscriber is dropped": testSlowSubscriberDropped,
		"late subscriber sees nothing": testNoReplay,
	} {
		t.Run(name, func(t *testing.T) { check(t, newBus) })
	}
}

func testDelivers(t *testing.T, newBus Factory) {
	b, ctx := newBus(t), t.Context()
	ch := subscribe(t, b, ctx)

	publish(t, b, ctx, "room", "thing.happened")
	got := receive(t, ch)
	if got.Kind != "thing.happened" {
		t.Errorf("Kind = %q, want thing.happened", got.Kind)
	}
	if len(got.Rooms) != 1 || got.Rooms[0] != "room" {
		t.Errorf("Rooms = %v, want [room]", got.Rooms)
	}
}

// Every subscriber gets every message: the bus is a broadcast, and deciding
// who cares is the hub's job, not this one's.
func testFanOut(t *testing.T, newBus Factory) {
	b, ctx := newBus(t), t.Context()
	channels := []<-chan bus.Message{subscribe(t, b, ctx), subscribe(t, b, ctx), subscribe(t, b, ctx)}

	publish(t, b, ctx, "room", "one")
	for i, ch := range channels {
		if got := receive(t, ch); got.Kind != "one" {
			t.Errorf("subscriber %d got %q, want one", i, got.Kind)
		}
	}
}

func testOrder(t *testing.T, newBus Factory) {
	b, ctx := newBus(t), t.Context()
	ch := subscribe(t, b, ctx)

	for _, kind := range []string{"a", "b", "c"} {
		publish(t, b, ctx, "room", kind)
	}
	for _, want := range []string{"a", "b", "c"} {
		if got := receive(t, ch); got.Kind != want {
			t.Fatalf("out of order: got %q, want %q", got.Kind, want)
		}
	}
}

// Sequence numbers are what let a subscriber tell "nothing happened" from
// "I missed something", so they must be contiguous for one keeping up.
func testSeq(t *testing.T, newBus Factory) {
	b, ctx := newBus(t), t.Context()
	ch := subscribe(t, b, ctx)

	for range 5 {
		publish(t, b, ctx, "room", "x")
	}
	previous := receive(t, ch).Seq
	for range 4 {
		next := receive(t, ch).Seq
		if next != previous+1 {
			t.Fatalf("Seq jumped from %d to %d with nothing dropped", previous, next)
		}
		previous = next
	}
}

func testContextEnds(t *testing.T, newBus Factory) {
	b := newBus(t)
	ctx, cancel := context.WithCancel(t.Context())
	ch := subscribe(t, b, ctx)
	cancel()

	select {
	case _, open := <-ch:
		if open {
			t.Error("channel delivered after its context was canceled")
		}
	case <-time.After(wait):
		t.Error("channel never closed after its context was canceled")
	}
}

// A subscriber that stops reading must not be able to stall the publisher,
// or one wedged client would take the system down.
func testSlowSubscriberDropped(t *testing.T, newBus Factory) {
	b, ctx := newBus(t), t.Context()
	slow := subscribe(t, b, ctx)
	keepingUp := subscribe(t, b, ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 5000 {
			if err := b.Publish(ctx, bus.Message{Rooms: []bus.Room{"room"}, Kind: "flood"}); err != nil {
				return
			}
			// Drain the healthy subscriber so only the slow one is behind.
			select {
			case <-keepingUp:
			default:
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(wait):
		t.Fatal("publishing blocked on a subscriber that stopped reading")
	}
	_ = slow
}

// The bus carries notification, not history. Anything missed is recovered
// from the store, not from here.
func testNoReplay(t *testing.T, newBus Factory) {
	b, ctx := newBus(t), t.Context()
	publish(t, b, ctx, "room", "before")

	ch := subscribe(t, b, ctx)
	publish(t, b, ctx, "room", "after")

	if got := receive(t, ch); got.Kind != "after" {
		t.Errorf("got %q; a new subscriber must not be replayed history", got.Kind)
	}
}

func subscribe(t *testing.T, b bus.Bus, ctx context.Context) <-chan bus.Message {
	t.Helper()
	ch, err := b.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	return ch
}

func publish(t *testing.T, b bus.Bus, ctx context.Context, room bus.Room, kind string) {
	t.Helper()
	if err := b.Publish(ctx, bus.Message{Rooms: []bus.Room{room}, Kind: kind}); err != nil && !errors.Is(err, bus.ErrClosed) {
		t.Fatalf("Publish: %v", err)
	}
}

func receive(t *testing.T, ch <-chan bus.Message) bus.Message {
	t.Helper()
	select {
	case m, open := <-ch:
		if !open {
			t.Fatal("channel closed while waiting for a message")
		}
		return m
	case <-time.After(wait):
		t.Fatal("timed out waiting for a message")
		return bus.Message{}
	}
}
