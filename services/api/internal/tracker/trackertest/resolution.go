package trackertest

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// The resolution gate is the rule that makes the tree mean something:
// closing a parent asserts everything beneath it is settled. An invariant
// that only holds when approached from one direction does not hold, so each
// of the three ways to falsify it is tested here.
func runResolution(t *testing.T, newStore Factory) {
	t.Run("a leaf resolves freely", func(t *testing.T) {
		s, _ := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		resolve(t, s, item)
	})

	t.Run("refuses to resolve over open work", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		parent := create(t, s, tracker.NewItem{})
		child(t, s, parent.ID)

		parent, err := s.Item(ctx, parent.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		doing, err := move(t, s, parent, "doing")
		if err != nil {
			t.Fatalf("open -> doing: %v", err)
		}
		note := tracker.Markdown("done")
		_, err = s.UpdateItem(ctx, doing.ID, doing.Version, tracker.Patch{
			Status: statusPtr("fixed"),
			Fields: map[tracker.FieldKey]*tracker.Value{"resolution": &note},
		})
		if !errors.Is(err, tracker.ErrUnresolvedDescendants) {
			t.Errorf("resolving over an open child = %v, want ErrUnresolvedDescendants", err)
		}
	})

	// Depth is unbounded, so the gate has to see all the way down, not just
	// at the direct children.
	t.Run("sees open work at any depth", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		root := create(t, s, tracker.NewItem{})
		mid := child(t, s, root.ID)
		leaf := child(t, s, mid.ID)

		resolve(t, s, leaf)
		if n, err := s.UnresolvedDescendants(ctx, root.ID); err != nil || n != 1 {
			t.Fatalf("UnresolvedDescendants = %d, %v; want 1 (mid is still open)", n, err)
		}

		root, _ = s.Item(ctx, root.ID)
		doing, err := move(t, s, root, "doing")
		if err != nil {
			t.Fatalf("open -> doing: %v", err)
		}
		note := tracker.Markdown("done")
		_, err = s.UpdateItem(ctx, doing.ID, doing.Version, tracker.Patch{
			Status: statusPtr("fixed"),
			Fields: map[tracker.FieldKey]*tracker.Value{"resolution": &note},
		})
		if !errors.Is(err, tracker.ErrUnresolvedDescendants) {
			t.Errorf("resolving over a deep open item = %v, want ErrUnresolvedDescendants", err)
		}
	})

	t.Run("resolves once the subtree is settled", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		parent := create(t, s, tracker.NewItem{})
		kid := child(t, s, parent.ID)

		resolve(t, s, kid)
		if n, err := s.UnresolvedDescendants(ctx, parent.ID); err != nil || n != 0 {
			t.Fatalf("UnresolvedDescendants = %d, %v; want 0", n, err)
		}
		parent, _ = s.Item(ctx, parent.ID)
		resolve(t, s, parent)
	})

	// Canceled counts as resolved, so an abandoned child must not hold its
	// parent open forever.
	t.Run("a canceled child does not hold the parent open", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		parent := create(t, s, tracker.NewItem{})
		kid := child(t, s, parent.ID)

		if _, err := move(t, s, kid, "dropped"); err != nil {
			t.Fatalf("open -> dropped: %v", err)
		}
		if n, err := s.UnresolvedDescendants(ctx, parent.ID); err != nil || n != 0 {
			t.Errorf("UnresolvedDescendants = %d, %v; want 0 for a canceled child", n, err)
		}
	})

	// Canceling a parent is held to the same rule as completing one: the
	// decision recorded in ADR-0004 is to refuse, not to cascade.
	t.Run("canceling is gated exactly like completing", func(t *testing.T) {
		s, _ := fixture(t, newStore)
		parent := create(t, s, tracker.NewItem{})
		child(t, s, parent.ID)

		_, err := move(t, s, parent, "dropped")
		if !errors.Is(err, tracker.ErrUnresolvedDescendants) {
			t.Errorf("canceling over an open child = %v, want ErrUnresolvedDescendants", err)
		}
	})

	// The second direction: work cannot be created beneath something that
	// has already closed, or the gate is false the moment it is added.
	t.Run("refuses new work under a resolved parent", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		parent := resolve(t, s, create(t, s, tracker.NewItem{}))

		_, err := s.CreateItem(ctx, tracker.NewItem{
			Type:   "bug",
			Parent: idPtr(parent.ID),
			Fields: map[tracker.FieldKey]tracker.Value{"summary": tracker.Text("late")},
		})
		if !errors.Is(err, tracker.ErrResolvedParent) {
			t.Errorf("CreateItem under a resolved parent = %v, want ErrResolvedParent", err)
		}
	})

	t.Run("refuses moving open work under a resolved parent", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		closed := resolve(t, s, create(t, s, tracker.NewItem{}))
		open := create(t, s, tracker.NewItem{})

		_, err := s.UpdateItem(ctx, open.ID, open.Version, tracker.Patch{Parent: idPtr(closed.ID)})
		if !errors.Is(err, tracker.ErrResolvedParent) {
			t.Errorf("reparenting open work under a resolved item = %v, want ErrResolvedParent", err)
		}
	})

	// The third direction: reopening beneath something already closed would
	// falsify the gate from below.
	t.Run("refuses reopening beneath a resolved ancestor", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		parent := create(t, s, tracker.NewItem{})
		kid := child(t, s, parent.ID)

		kid = resolve(t, s, kid)
		parent, _ = s.Item(ctx, parent.ID)
		resolve(t, s, parent)

		_, err := s.UpdateItem(ctx, kid.ID, kid.Version, tracker.Patch{Status: statusPtr("open")})
		if !errors.Is(err, tracker.ErrResolvedParent) {
			t.Errorf("reopening under a resolved parent = %v, want ErrResolvedParent", err)
		}
	})

	t.Run("counts nothing beneath a leaf", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if n, err := s.UnresolvedDescendants(ctx, item.ID); err != nil || n != 0 {
			t.Errorf("UnresolvedDescendants on a leaf = %d, %v; want 0", n, err)
		}
	})

	t.Run("reports a missing item", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		if _, err := s.UnresolvedDescendants(ctx, "ghost"); !errors.Is(err, tracker.ErrNotFound) {
			t.Errorf("UnresolvedDescendants = %v, want ErrNotFound", err)
		}
	})
}
