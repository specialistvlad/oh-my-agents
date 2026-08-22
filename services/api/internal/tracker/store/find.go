package store

import (
	"context"
	"slices"
	"strings"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// FindItems implements [tracker.ItemFinder].
func (s *Store) FindItems(ctx context.Context, q tracker.Query) (tracker.ItemPage, error) {
	if err := ctx.Err(); err != nil {
		return tracker.ItemPage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched []tracker.Item
	for _, item := range s.items {
		if s.matches(item, q) {
			matched = append(matched, clone(item))
		}
	}
	s.sortItems(matched, q.Sort)
	return paginate(matched, q.Page)
}

// matches applies every criterion; they are ANDed, and each is satisfied by
// membership rather than by an expression.
func (s *Store) matches(item tracker.Item, q tracker.Query) bool {
	switch {
	case len(q.Types) > 0 && !slices.Contains(q.Types, item.Type):
		return false
	case len(q.Statuses) > 0 && !slices.Contains(q.Statuses, item.Status):
		return false
	case len(q.Categories) > 0 && !slices.Contains(q.Categories, s.categoryOf(item)):
		return false
	case q.Roots && item.Parent != nil:
		return false
	case q.Parent != nil && (item.Parent == nil || *item.Parent != *q.Parent):
		return false
	case q.UpdatedSince != nil && item.UpdatedAt.Before(*q.UpdatedSince):
		return false
	case q.Subtree != nil && !s.isBelow(item, *q.Subtree):
		return false
	}
	for _, m := range q.Fields {
		v, held := item.Fields[m.Field]
		if !held || !v.Equal(m.Equals) {
			return false
		}
	}
	return true
}

// isBelow reports whether item sits anywhere beneath root, excluding root.
func (s *Store) isBelow(item tracker.Item, root tracker.ItemID) bool {
	if item.ID == root || item.Parent == nil {
		return false
	}
	for _, ancestor := range s.ancestorsOf(*item.Parent) {
		if ancestor.ID == root {
			return true
		}
	}
	return false
}

// categoryOf reads an item's category, which is the axis every generic
// question about progress is asked along.
func (s *Store) categoryOf(item tracker.Item) tracker.StatusCategory {
	t, ok := s.schema.Type(item.Type)
	if !ok {
		return ""
	}
	st, ok := t.Status(item.Status)
	if !ok {
		return ""
	}
	return st.Category
}

// sortItems orders a result set. ID breaks every tie, so the order is total
// and a page boundary cannot land in the middle of an ambiguous run.
func (s *Store) sortItems(items []tracker.Item, sort tracker.Sort) {
	slices.SortStableFunc(items, func(a, b tracker.Item) int {
		if c := compareBy(a, b, sort.By); c != 0 {
			if sort.Desc {
				return -c
			}
			return c
		}
		return strings.Compare(string(a.ID), string(b.ID))
	})
}

func compareBy(a, b tracker.Item, by tracker.SortKey) int {
	switch by {
	case tracker.SortUpdatedAt:
		return a.UpdatedAt.Compare(b.UpdatedAt)
	case tracker.SortTitle:
		return strings.Compare(a.Title, b.Title)
	case tracker.SortCreatedAt:
		return a.CreatedAt.Compare(b.CreatedAt)
	default:
		return a.CreatedAt.Compare(b.CreatedAt)
	}
}
