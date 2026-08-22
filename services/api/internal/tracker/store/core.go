// Package memory is an in-memory [tracker.Store].
//
// It is the fake ADR-0002 requires, and it is held to the same guarantees as
// any other adapter by the trackertest suite — including the enforcement
// ADR-0005 puts on adapters rather than on a layer above them. Everything a
// filesystem or SQL store must refuse, this refuses too.
package store

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Store keeps everything in maps behind one mutex. Reads and writes are both
// short, so a single lock is simpler than being clever and fast enough for
// what a fake is for.
type Store struct {
	mu       sync.RWMutex
	disk     Persistence
	announce func(context.Context, tracker.Event)
	clock    tracker.Clock
	ids      tracker.IDGenerator
	schema   tracker.Schema
	items    map[tracker.ItemID]tracker.Item
	comments map[tracker.CommentID]tracker.Comment
	links    []tracker.Link
	events   []tracker.Event
	seq      uint64
}

// Deps are the ambient dependencies. Each defaults to something sensible, so
// a caller that does not care can pass the zero value.
type Deps struct {
	Clock tracker.Clock
	IDs   tracker.IDGenerator

	// Persistence is where state survives the process. Nil keeps nothing,
	// which is what a fake wants.
	Persistence Persistence

	// Announce is told about every event, after it has been persisted and
	// applied. Nil tells nobody.
	//
	// A hook rather than a decorator over [tracker.Store]: the store already
	// computes the exact event, and wrapping eighteen methods to forward
	// what one function can carry would be all cost and no clarity
	// (ADR-0001).
	Announce func(ctx context.Context, e tracker.Event)
}

// New returns a store holding whatever its persistence already had.
func New(ctx context.Context, d Deps) (*Store, error) {
	if d.Clock == nil {
		d.Clock = tracker.SystemClock{}
	}
	if d.IDs == nil {
		d.IDs = tracker.RandomIDs{}
	}
	if d.Persistence == nil {
		d.Persistence = nothing{}
	}
	s := &Store{
		clock:    d.Clock,
		ids:      d.IDs,
		disk:     d.Persistence,
		announce: d.Announce,
		items:    make(map[tracker.ItemID]tracker.Item),
		comments: make(map[tracker.CommentID]tracker.Comment),
	}
	held, err := d.Persistence.Load(ctx)
	if err != nil {
		return nil, err
	}
	s.schema = held.Schema
	for _, item := range held.Items {
		s.items[item.ID] = item
	}
	for _, c := range held.Comments {
		s.comments[c.ID] = c
	}
	s.links = held.Links
	s.events = held.Events
	for _, e := range held.Events {
		if e.Seq > s.seq {
			s.seq = e.Seq
		}
	}
	return s, nil
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
		if existing.ID != t.ID {
			next.Types = append(next.Types, existing)
		}
	}
	next.Types = append(next.Types, cloneType(t))

	for _, item := range s.items {
		if item.Type != t.ID {
			continue
		}
		if err := next.ValidateItem(item); err != nil {
			return fmt.Errorf("%w: %q would invalidate item %q: %w",
				tracker.ErrInvalidSchema, t.ID, item.ID, err)
		}
	}
	if err := s.disk.SaveType(ctx, t); err != nil {
		return err
	}
	s.schema = next
	return nil
}

// DeleteItemType implements [tracker.SchemaWriter]. A type still in use
// cannot be removed, for the same reason.
func (s *Store) DeleteItemType(ctx context.Context, key tracker.TypeID) error {
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
		if t.ID != key {
			kept = append(kept, t)
		}
	}
	if err := s.disk.DeleteType(ctx, key); err != nil {
		return err
	}
	s.schema.Types = kept
	return nil
}

// cloneSchema deep-copies the configuration. Copying only the outer slice
// would leave every type's fields, statuses and transitions shared with the
// store, so a caller could rewrite the schema through a value it was handed.
// Callers hold the read lock.
func (s *Store) cloneSchema() tracker.Schema {
	out := tracker.Schema{Types: make([]tracker.ItemType, 0, len(s.schema.Types))}
	for _, t := range s.schema.Types {
		out.Types = append(out.Types, cloneType(t))
	}
	return out
}

// cloneType copies a type and everything hanging off it.
func cloneType(t tracker.ItemType) tracker.ItemType {
	out := t
	out.Fields = make([]tracker.FieldDef, 0, len(t.Fields))
	for _, f := range t.Fields {
		f.Options = slices.Clone(f.Options)
		f.ItemTypes = slices.Clone(f.ItemTypes)
		if f.Default != nil {
			def := *f.Default
			f.Default = &def
		}
		out.Fields = append(out.Fields, f)
	}
	out.Statuses = slices.Clone(t.Statuses)
	out.Transitions = make([]tracker.Transition, 0, len(t.Transitions))
	for _, tr := range t.Transitions {
		tr.RequiredFields = slices.Clone(tr.RequiredFields)
		out.Transitions = append(out.Transitions, tr)
	}
	return out
}
