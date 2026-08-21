// Package bus is the seam between processes: what happened over there, so a
// client connected here hears about it.
//
// It carries notification, not truth (ADR-0008). A message is a low-latency
// hint; correctness comes from sequence numbers, because a subscriber that
// falls behind is dropped from rather than blocking the publisher. A gap in
// [Message.Seq] is the subscriber's signal that it missed something and must
// re-read from the store.
//
// Two implementations, one conformance suite in bustest: [Memory], which
// needs nothing installed and is the default, and a Valkey one for when more
// than one process is running. Nothing above this package knows which it has.
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
	// message. It exists so a subscriber can tell "nothing has happened"
	// from "I missed something": a gap means messages were dropped and the
	// subscriber must resynchronise.
	//
	// It is the bus's own ordering and is not a resume token. A client
	// resumes from the sequence of the event log the payload came from,
	// which travels inside Data.
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
