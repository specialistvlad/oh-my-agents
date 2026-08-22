package store

import (
	"context"
	"fmt"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Reorder implements [tracker.ItemWriter].
//
// It states no version and bumps none (ADR-0013). The rank it mints is derived
// from the neighbors the caller named, which is what keeps the key algebra
// inside the tracker package.
func (s *Store) Reorder(ctx context.Context, id tracker.ItemID, after, before *tracker.ItemID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	item, err := s.load(id)
	if err != nil {
		return err
	}
	prev, err := s.rankOf(after)
	if err != nil {
		return err
	}
	next, err := s.rankOf(before)
	if err != nil {
		return err
	}
	rank, err := tracker.Between(prev, next)
	if err != nil {
		return fmt.Errorf("%w: %w", tracker.ErrInvalidSchema, err)
	}
	item.Rank = rank

	if err := s.disk.SaveItem(ctx, item); err != nil {
		return err
	}
	s.items[id] = item
	s.emit(ctx, id, tracker.EventItemReordered, item.UpdatedBy, s.clock.Now(), nil)
	return nil
}

// rankOf reads a neighbor's rank. A nil neighbor is an open end — the start
// of the order or the end of it — which is why it yields the empty rank rather
// than an error.
func (s *Store) rankOf(id *tracker.ItemID) (tracker.Rank, error) {
	if id == nil {
		return "", nil
	}
	neighbor, err := s.load(*id)
	if err != nil {
		return "", err
	}
	return neighbor.Rank, nil
}

// lastRank is the rank of the item currently at the end of the project, or the
// empty rank when there is none. New items go after it: appending is the only
// answer to "where does this go" that does not surprise someone (ADR-0013).
//
// Callers hold the lock.
func (s *Store) lastRank() tracker.Rank {
	var last tracker.Rank
	for _, item := range s.items {
		if item.Rank > last {
			last = item.Rank
		}
	}
	return last
}
