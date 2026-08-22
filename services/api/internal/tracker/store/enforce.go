package store

import (
	"fmt"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// The tree rules live here because they are the ones a schema cannot answer:
// each has to look at items other than the one being written.
//
// The resolution gate is guarded from three directions, because an invariant
// that only holds when approached one way does not hold. An item may not
// resolve while work beneath it is open; unresolved work may not be moved or
// created under a resolved parent; and an item may not reopen beneath an
// ancestor that has already resolved. Enforcing only the first would let the
// other two produce exactly the state the gate exists to prevent.

// checkStructure applies the tree rules to one update. Callers hold the write
// lock and have already passed the patch through the schema.
func (s *Store) checkStructure(current, next tracker.Item, p tracker.Patch) error {
	if p.Parent != nil || p.ClearParent {
		if next.Parent != nil {
			if err := s.checkAttach(next, *next.Parent); err != nil {
				return err
			}
		}
	}
	if p.Status == nil || *p.Status == current.Status {
		return nil
	}
	return s.checkResolution(next)
}

// checkAttach vets a parent link: the parent has to exist, the link must not
// close a cycle, and unresolved work must not hang off a resolved parent.
func (s *Store) checkAttach(child tracker.Item, parentID tracker.ItemID) error {
	parent, ok := s.items[parentID]
	if !ok {
		return fmt.Errorf("%w: parent %q", tracker.ErrNotFound, parentID)
	}
	if parentID == child.ID {
		return fmt.Errorf("%w: %q cannot parent itself", tracker.ErrCycle, child.ID)
	}
	for _, ancestor := range s.ancestorsOf(parentID) {
		if ancestor.ID == child.ID {
			return fmt.Errorf("%w: %q is already an ancestor of %q",
				tracker.ErrCycle, child.ID, parentID)
		}
	}
	if s.resolved(child) {
		return nil
	}
	if s.resolved(parent) {
		return fmt.Errorf("%w: %q is %s, so it cannot take unresolved work",
			tracker.ErrResolvedParent, parentID, parent.Status)
	}
	return nil
}

// checkResolution vets a status move in both directions: resolving needs a
// settled subtree, and reopening needs no resolved ancestor.
func (s *Store) checkResolution(next tracker.Item) error {
	if s.resolved(next) {
		if open := s.unresolvedBelow(next.ID); open > 0 {
			return fmt.Errorf("%w: %q has %d unresolved below it",
				tracker.ErrUnresolvedDescendants, next.ID, open)
		}
		return nil
	}
	if next.Parent == nil {
		return nil
	}
	for _, ancestor := range s.ancestorsOf(*next.Parent) {
		if s.resolved(ancestor) {
			return fmt.Errorf("%w: %q is %s, so %q cannot reopen beneath it",
				tracker.ErrResolvedParent, ancestor.ID, ancestor.Status, next.ID)
		}
	}
	return nil
}

// resolved reports whether an item sits in a resolved category. An item whose
// type or status has gone missing counts as unresolved: the safe reading,
// since it keeps a parent from closing over work nobody can account for.
func (s *Store) resolved(item tracker.Item) bool {
	t, ok := s.schema.Type(item.Type)
	if !ok {
		return false
	}
	st, ok := t.Status(item.Status)
	if !ok {
		return false
	}
	return st.Category.Resolved()
}

// unresolvedBelow counts unresolved items anywhere beneath id.
func (s *Store) unresolvedBelow(id tracker.ItemID) int {
	open := 0
	for _, child := range s.childrenOf(id) {
		if !s.resolved(child) {
			open++
		}
		open += s.unresolvedBelow(child.ID)
	}
	return open
}

// childrenOf returns the direct children of id, in no particular order.
func (s *Store) childrenOf(id tracker.ItemID) []tracker.Item {
	var out []tracker.Item
	for _, item := range s.items {
		if item.Parent != nil && *item.Parent == id {
			out = append(out, item)
		}
	}
	return out
}

// ancestorsOf walks up from id inclusive, nearest first. It stops if it ever
// revisits an item, so a cycle that somehow reached storage cannot hang a
// read while the caller holds the lock.
func (s *Store) ancestorsOf(id tracker.ItemID) []tracker.Item {
	var out []tracker.Item
	seen := make(map[tracker.ItemID]struct{})
	for {
		item, ok := s.items[id]
		if !ok {
			return out
		}
		if _, looped := seen[id]; looped {
			return out
		}
		seen[id] = struct{}{}
		out = append(out, item)
		if item.Parent == nil {
			return out
		}
		id = *item.Parent
	}
}
