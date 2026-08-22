package store

import (
	"context"
	"fmt"
	"maps"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Item implements [tracker.ItemReader].
func (s *Store) Item(ctx context.Context, id tracker.ItemID) (tracker.Item, error) {
	if err := ctx.Err(); err != nil {
		return tracker.Item{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.load(id)
}

// CreateItem implements [tracker.ItemWriter]. A new item enters its type's
// initial status without consulting the transition graph.
func (s *Store) CreateItem(ctx context.Context, n tracker.NewItem) (tracker.Item, error) {
	if err := ctx.Err(); err != nil {
		return tracker.Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.schema.Type(n.Type)
	if !ok {
		return tracker.Item{}, fmt.Errorf("%w: %q", tracker.ErrUnknownType, n.Type)
	}
	if err := s.schema.ValidateNew(n); err != nil {
		return tracker.Item{}, err
	}
	// Appending is the only answer to "where does a new item go" that does
	// not surprise someone (ADR-0013).
	rank, err := tracker.Between(s.lastRank(), "")
	if err != nil {
		return tracker.Item{}, err
	}
	now := s.clock.Now()
	item := tracker.Item{
		ID:        tracker.ItemID(s.ids.NewID()),
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Status:    t.Initial,
		Parent:    n.Parent,
		Fields:    t.ApplyDefaults(n.Fields),
		Rank:      rank,
		CreatedBy: n.Author,
		CreatedAt: now,
		UpdatedBy: n.Author,
		UpdatedAt: now,
		Version:   1,
	}
	if n.Parent != nil {
		if err := s.checkAttach(item, *n.Parent); err != nil {
			return tracker.Item{}, err
		}
	}
	if err := s.disk.SaveItem(ctx, item); err != nil {
		return tracker.Item{}, err
	}
	s.items[item.ID] = item
	s.emit(ctx, item.ID, tracker.EventItemCreated, n.Author, now, nil)
	return clone(item), nil
}

// UpdateItem implements [tracker.ItemWriter]. The expected version must match
// what is stored, so two agents editing one item cannot overwrite each other
// unnoticed.
func (s *Store) UpdateItem(
	ctx context.Context, id tracker.ItemID, expected tracker.Version, p tracker.Patch,
) (tracker.Item, error) {
	if err := ctx.Err(); err != nil {
		return tracker.Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.load(id)
	if err != nil {
		return tracker.Item{}, err
	}
	if current.Version != expected {
		return tracker.Item{}, fmt.Errorf("%w: item %q is at version %d, not %d",
			tracker.ErrVersionConflict, id, current.Version, expected)
	}
	if err := s.schema.ValidatePatch(current, p); err != nil {
		return tracker.Item{}, err
	}
	next := applyPatch(current, p)
	if err := s.checkStructure(current, next, p); err != nil {
		return tracker.Item{}, err
	}
	next.Version = current.Version + 1
	next.UpdatedBy = p.Author
	next.UpdatedAt = s.clock.Now()

	if err := s.disk.SaveItem(ctx, next); err != nil {
		return tracker.Item{}, err
	}
	s.items[id] = next
	s.emitChanges(ctx, current, next, p.Author)
	return clone(next), nil
}

// DeleteItem implements [tracker.ItemWriter]. An item with children is not
// deleted: doing so would either orphan them or remove a subtree silently,
// and neither is a decision this call should make by itself.
func (s *Store) DeleteItem(
	ctx context.Context, id tracker.ItemID, expected tracker.Version, by tracker.ActorRef,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.load(id)
	if err != nil {
		return err
	}
	if current.Version != expected {
		return fmt.Errorf("%w: item %q is at version %d, not %d",
			tracker.ErrVersionConflict, id, current.Version, expected)
	}
	for _, child := range s.items {
		if child.Parent != nil && *child.Parent == id {
			return fmt.Errorf("%w: %q still parents %q", tracker.ErrHasChildren, id, child.ID)
		}
	}
	if err := s.disk.DeleteItem(ctx, id); err != nil {
		return err
	}
	delete(s.items, id)
	s.emit(ctx, id, tracker.EventItemDeleted, by, s.clock.Now(), nil)
	return nil
}

// load reads one item under whichever lock the caller holds.
func (s *Store) load(id tracker.ItemID) (tracker.Item, error) {
	item, ok := s.items[id]
	if !ok {
		return tracker.Item{}, fmt.Errorf("%w: item %q", tracker.ErrNotFound, id)
	}
	return clone(item), nil
}

// applyPatch produces the item a patch results in. The schema has already
// checked it, so nothing here can fail.
func applyPatch(current tracker.Item, p tracker.Patch) tracker.Item {
	next := clone(current)
	if p.Title != nil {
		next.Title = *p.Title
	}
	if p.Body != nil {
		next.Body = *p.Body
	}
	if p.Status != nil {
		next.Status = *p.Status
	}
	switch {
	case p.ClearParent:
		next.Parent = nil
	case p.Parent != nil:
		parent := *p.Parent
		next.Parent = &parent
	}
	for key, v := range p.Fields {
		if v == nil {
			delete(next.Fields, key)
			continue
		}
		next.Fields[key] = *v
	}
	return next
}

// clone copies an item deeply enough that a caller cannot reach the store's
// own maps or pointers.
func clone(item tracker.Item) tracker.Item {
	out := item
	out.Fields = maps.Clone(item.Fields)
	if out.Fields == nil {
		out.Fields = make(map[tracker.FieldID]tracker.Value)
	}
	if item.Parent != nil {
		parent := *item.Parent
		out.Parent = &parent
	}
	return out
}
