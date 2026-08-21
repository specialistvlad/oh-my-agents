// Package memory is an in-memory [tracker.Store].
//
// It is the fake ADR-0002 requires, and it is held to the same guarantees as
// any other adapter by the trackertest suite — including the enforcement
// ADR-0005 puts on adapters rather than on a layer above them. Everything a
// filesystem or SQL store must refuse, this refuses too.
package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Store keeps everything in maps behind one mutex. Reads and writes are both
// short, so a single lock is simpler than being clever and fast enough for
// what a fake is for.
type Store struct {
	mu       sync.RWMutex
	clock    tracker.Clock
	ids      tracker.IDGenerator
	schema   tracker.Schema
	items    map[tracker.ItemID]tracker.Item
	comments map[tracker.CommentID]tracker.Comment
	links    []tracker.Link
	events   []tracker.Event
	seq      uint64
}

// Deps are the ambient dependencies. Both default to the real thing, so a
// caller that does not care can pass the zero value.
type Deps struct {
	Clock tracker.Clock
	IDs   tracker.IDGenerator
}

// New returns an empty store with no types configured.
func New(d Deps) *Store {
	if d.Clock == nil {
		d.Clock = tracker.SystemClock{}
	}
	if d.IDs == nil {
		d.IDs = tracker.RandomIDs{}
	}
	return &Store{
		clock:    d.Clock,
		ids:      d.IDs,
		items:    make(map[tracker.ItemID]tracker.Item),
		comments: make(map[tracker.CommentID]tracker.Comment),
	}
}

// Schema implements [tracker.SchemaReader].
func (s *Store) Schema(ctx context.Context) (tracker.Schema, error) {
	if err := ctx.Err(); err != nil {
		return tracker.Schema{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cloneSchema(), nil
}

// PutItemType implements [tracker.SchemaWriter]. The type is validated whole
// before it lands, and a change that would invalidate stored items is
// refused — a schema that contradicts its own data is not a state worth
// being able to reach.
func (s *Store) PutItemType(ctx context.Context, t tracker.ItemType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := t.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	next := tracker.Schema{Types: make([]tracker.ItemType, 0, len(s.schema.Types)+1)}
	for _, existing := range s.schema.Types {
		if existing.Key != t.Key {
			next.Types = append(next.Types, existing)
		}
	}
	next.Types = append(next.Types, t)

	for _, item := range s.items {
		if item.Type != t.Key {
			continue
		}
		if err := next.ValidateItem(item); err != nil {
			return fmt.Errorf("%w: %q would invalidate item %q: %w",
				tracker.ErrInvalidSchema, t.Key, item.ID, err)
		}
	}
	s.schema = next
	return nil
}

// DeleteItemType implements [tracker.SchemaWriter]. A type still in use
// cannot be removed, for the same reason.
func (s *Store) DeleteItemType(ctx context.Context, key tracker.TypeKey) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schema.Type(key); !ok {
		return fmt.Errorf("%w: %q", tracker.ErrUnknownType, key)
	}
	for _, item := range s.items {
		if item.Type == key {
			return fmt.Errorf("%w: type %q is still used by item %q",
				tracker.ErrInvalidSchema, key, item.ID)
		}
	}
	kept := make([]tracker.ItemType, 0, len(s.schema.Types))
	for _, t := range s.schema.Types {
		if t.Key != key {
			kept = append(kept, t)
		}
	}
	s.schema.Types = kept
	return nil
}

// cloneSchema copies enough that a caller cannot reach into the store's own
// slices. Callers hold the read lock.
func (s *Store) cloneSchema() tracker.Schema {
	out := tracker.Schema{Types: make([]tracker.ItemType, len(s.schema.Types))}
	copy(out.Types, s.schema.Types)
	return out
}
