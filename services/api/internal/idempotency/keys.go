// Package idempotency remembers what a command already did, so replaying it
// does not do it twice.
//
// A bidirectional socket needs this in a way a request/response API does not.
// A client that never saw an acknowledgement cannot know whether its command
// arrived, and its only sensible move is to send it again after reconnecting.
// Without a memory of what was already applied, a dropped acknowledgement
// silently duplicates the work — and for anything that is not naturally
// idempotent, a delete most of all, the replay reports a failure for
// something that in fact succeeded.
package idempotency

import (
	"sync"
	"time"
)

// Outcome is what a command produced, kept so a replay can be answered with
// the original result rather than by doing the work again.
type Outcome struct {
	Err error
}

// Keys remembers outcomes for a while.
//
// The window is deliberately short. It only has to outlive a reconnect, and a
// key kept forever is a memory leak with extra steps.
type Keys struct {
	mu      sync.Mutex
	seen    map[string]entry
	ttl     time.Duration
	limit   int
	now     func() time.Time
	lastGC  time.Time
	gcEvery time.Duration
}

type entry struct {
	outcome Outcome
	at      time.Time
}

// Options configure a [Keys]. The zero value gives sensible defaults.
type Options struct {
	// TTL is how long an outcome is remembered. Default five minutes: long
	// enough to cover a reconnect, short enough to forget.
	TTL time.Duration
	// Limit caps how many are held, so a client inventing keys cannot grow
	// the map without bound. Default 4096.
	Limit int
	// Now is the clock, injected so tests need not sleep.
	Now func() time.Time
}

// New returns an empty memory.
func New(o Options) *Keys {
	if o.TTL <= 0 {
		o.TTL = 5 * time.Minute
	}
	if o.Limit <= 0 {
		o.Limit = 4096
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return &Keys{
		seen:    make(map[string]entry),
		ttl:     o.TTL,
		limit:   o.Limit,
		now:     o.Now,
		gcEvery: o.TTL / 2,
	}
}

// Recall returns what a key produced before, if it is still remembered.
// An empty key is never remembered: a caller that supplies none is asking for
// the command to run every time.
func (k *Keys) Recall(key string) (Outcome, bool) {
	if key == "" {
		return Outcome{}, false
	}
	k.mu.Lock()
	defer k.mu.Unlock()

	k.collect()
	found, ok := k.seen[key]
	if !ok || k.now().Sub(found.at) > k.ttl {
		return Outcome{}, false
	}
	return found.outcome, true
}

// Remember records what a key produced.
func (k *Keys) Remember(key string, outcome Outcome) {
	if key == "" {
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()

	k.collect()
	// At the cap, forget everything rather than evict cleverly. Losing the
	// memory costs a duplicated command; an LRU here would be machinery
	// nobody can justify for a map that exists to survive a reconnect.
	if len(k.seen) >= k.limit {
		clear(k.seen)
	}
	k.seen[key] = entry{outcome: outcome, at: k.now()}
}

// collect drops expired entries, at most every half-TTL so that a busy
// connection is not walking the map on every command. Callers hold the lock.
func (k *Keys) collect() {
	now := k.now()
	if now.Sub(k.lastGC) < k.gcEvery {
		return
	}
	k.lastGC = now
	for key, held := range k.seen {
		if now.Sub(held.at) > k.ttl {
			delete(k.seen, key)
		}
	}
}
