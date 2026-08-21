// Package bus is the seam between processes: what happened over there, so a
// client connected here hears about it.
//
// It carries notification, not truth (ADR-0008). A message is a low-latency
// hint; correctness comes from sequence numbers, because a subscriber that
// falls behind is dropped from rather than allowed to block the publisher. A
// gap in [Message.Seq] tells a subscriber it missed something — and the only
// thing it ever does about that is fetch the current state of what it cares
// about again. Nothing is replayed, here or anywhere.
//
// [Memory] is the one implementation: in-process, needing nothing installed.
// A networked one for running several processes is deliberately not built yet
// (ADR-0008); the port and the conformance suite in bustest exist so that
// adding it is an implementation rather than a rewrite.
package bus

import (
	"context"
	"encoding/json"
	"errors"
)

// Room is an address subscribers care about. The bus does not interpret one;
// it carries them so a hub can decide which connections a message concerns.
type Room string

// Message is one thing that happened.
type Message struct {
	// Rooms this message belongs to. A message with none goes nowhere,
	// which is a caller error rather than a broadcast.
	Rooms []Room

	// Seq is assigned by the bus at publish time and increases by one per
	// message. Its only job is to let a subscriber tell "nothing has
	// happened" from "I missed something": a gap means messages were
	// dropped.
	//
	// It is not a resume token and cannot be used as one. Nothing can be
	// replayed from it — a subscriber that sees a gap refetches current
	// state, it does not ask what it missed.
	Seq uint64

	Kind string
	Data json.RawMessage
}

// ErrClosed is a publish or subscribe on a bus that has been shut down.
var ErrClosed = errors.New("bus: closed")

// Publisher announces that something happened.
type Publisher interface {
	Publish(ctx context.Context, m Message) error
}

// Subscriber receives what others announced.
//
// The channel is closed when the subscription's context is done or the bus
// shuts down, so a range over it terminates without any other signal. A
// subscriber that stops reading is dropped from rather than allowed to block
// everyone else — the cost of falling behind is a gap in [Message.Seq], and
// noticing that is the subscriber's job.
type Subscriber interface {
	Subscribe(ctx context.Context) (<-chan Message, error)
}

// Bus is what an implementation asserts against:
//
//	var _ bus.Bus = (*bus.Memory)(nil)
//
// Consumers take [Publisher] or [Subscriber], not this.
type Bus interface {
	Publisher
	Subscriber
}
