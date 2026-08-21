package bus

import (
	"context"
	"sync"
)

// backlog is how many messages a subscriber may fall behind before it starts
// losing them. Deep enough to absorb a burst, shallow enough that a wedged
// subscriber is noticed quickly rather than growing without bound.
const backlog = 256

// Memory is a bus within one process. It is the default: nothing to install,
// nothing to run, and correct for a single process.
//
// Delivery is best-effort by design. A subscriber that stops reading has
// messages dropped rather than stalling the publisher, and detects that as a
// gap in [Message.Seq]. That is the same contract a networked bus offers, so
// code written against this one does not change when Valkey is switched on.
type Memory struct {
	mu     sync.Mutex
	subs   map[chan Message]struct{}
	seq    uint64
	closed bool
}

// NewMemory returns an empty bus.
func NewMemory() *Memory {
	return &Memory{subs: make(map[chan Message]struct{})}
}

// Publish implements [Publisher]. It never blocks on a slow subscriber.
func (b *Memory) Publish(ctx context.Context, m Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrClosed
	}
	b.seq++
	m.Seq = b.seq
	for ch := range b.subs {
		select {
		case ch <- m:
		default: // subscriber is behind; it will see the gap in Seq
		}
	}
	return nil
}

// Subscribe implements [Subscriber].
func (b *Memory) Subscribe(ctx context.Context) (<-chan Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil, ErrClosed
	}
	ch := make(chan Message, backlog)
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	// One goroutine per subscription, ending with its context. Removing the
	// channel before closing it means Publish can never send on a closed
	// channel, which is the only way this could panic.
	go func() {
		<-ctx.Done()
		b.remove(ch)
	}()
	return ch, nil
}

// Close ends every subscription and refuses further use.
func (b *Memory) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true
	for ch := range b.subs {
		delete(b.subs, ch)
		close(ch)
	}
	return nil
}

func (b *Memory) remove(ch chan Message) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, live := b.subs[ch]; !live {
		return // already gone, via Close
	}
	delete(b.subs, ch)
	close(ch)
}
