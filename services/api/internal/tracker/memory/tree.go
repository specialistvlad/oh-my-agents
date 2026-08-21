package memory

import (
	"context"
	"slices"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// UnresolvedDescendants implements [tracker.SubtreeReader]. Zero means every
// item beneath this one is settled, which is the condition a parent must meet
// before it may resolve.
func (s *Store) UnresolvedDescendants(ctx context.Context, id tracker.ItemID) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.load(id); err != nil {
		return 0, err
	}
	return s.unresolvedBelow(id), nil
}

// Ancestors implements [tracker.SubtreeReader], root first and excluding the
// item itself.
func (s *Store) Ancestors(ctx context.Context, id tracker.ItemID) ([]tracker.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, err := s.load(id)
	if err != nil {
		return nil, err
	}
	if item.Parent == nil {
		return nil, nil
	}
	up := s.ancestorsOf(*item.Parent)
	out := make([]tracker.Item, 0, len(up))
	for _, a := range up {
		out = append(out, clone(a))
	}
	slices.Reverse(out)
	return out, nil
}
