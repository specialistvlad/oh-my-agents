package trackertest

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func runVersioning(t *testing.T, newStore Factory) {
	t.Run("bumps the version on every write", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		title := "renamed"
		next, err := s.UpdateItem(ctx, item.ID, item.Version, tracker.Patch{Title: &title})
		if err != nil {
			t.Fatalf("UpdateItem: %v", err)
		}
		if next.Version != item.Version+1 {
			t.Errorf("Version = %d, want %d", next.Version, item.Version+1)
		}
	})

	// Two agents on one item is the normal case, not the edge case.
	t.Run("refuses a stale write", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		first := "first"
		if _, err := s.UpdateItem(ctx, item.ID, item.Version, tracker.Patch{Title: &first}); err != nil {
			t.Fatalf("first write: %v", err)
		}
		second := "second"
		_, err := s.UpdateItem(ctx, item.ID, item.Version, tracker.Patch{Title: &second})
		if !errors.Is(err, tracker.ErrVersionConflict) {
			t.Errorf("second write at a stale version = %v, want ErrVersionConflict", err)
		}
		got, err := s.Item(ctx, item.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if got.Title != "first" {
			t.Errorf("Title = %q; the losing write should not have landed", got.Title)
		}
	})

	t.Run("refuses a stale delete", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if err := s.DeleteItem(ctx, item.ID, item.Version+1, human("vk")); !errors.Is(err, tracker.ErrVersionConflict) {
			t.Errorf("DeleteItem = %v, want ErrVersionConflict", err)
		}
	})
}

func runWorkflow(t *testing.T, newStore Factory) {
	t.Run("allows a declared transition", func(t *testing.T) {
		s, _ := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if _, err := move(t, s, item, StatusDoing); err != nil {
			t.Errorf("open -> doing: %v", err)
		}
	})

	t.Run("refuses an undeclared transition", func(t *testing.T) {
		s, _ := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		_, err := move(t, s, item, StatusFixed) // only reachable via doing
		if !errors.Is(err, tracker.ErrTransitionNotAllowed) {
			t.Errorf("open -> fixed = %v, want ErrTransitionNotAllowed", err)
		}
	})

	t.Run("refuses an unknown status", func(t *testing.T) {
		s, _ := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if _, err := move(t, s, item, "invented"); !errors.Is(err, tracker.ErrUnknownStatus) {
			t.Errorf("move to an unknown status = %v, want ErrUnknownStatus", err)
		}
	})

	t.Run("enforces a transition's required fields", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		doing, err := move(t, s, item, StatusDoing)
		if err != nil {
			t.Fatalf("open -> doing: %v", err)
		}
		if _, err := move(t, s, doing, StatusFixed); !errors.Is(err, tracker.ErrFieldRequired) {
			t.Errorf("doing -> fixed without a resolution = %v, want ErrFieldRequired", err)
		}

		// Setting the field and moving in one write is allowed: the
		// requirement is judged against the outcome.
		note := tracker.Markdown("fixed it")
		_, err = s.UpdateItem(ctx, doing.ID, doing.Version, tracker.Patch{
			Status: statusPtr(StatusFixed),
			Fields: map[tracker.FieldID]*tracker.Value{FieldResolution: &note},
		})
		if err != nil {
			t.Errorf("setting the resolution and moving together: %v", err)
		}
	})
}

func runHierarchy(t *testing.T, newStore Factory) {
	t.Run("nests to arbitrary depth", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		deepest := create(t, s, tracker.NewItem{})
		for range 12 {
			deepest = child(t, s, deepest.ID)
		}
		ancestors, err := s.Ancestors(ctx, deepest.ID)
		if err != nil {
			t.Fatalf("Ancestors: %v", err)
		}
		if len(ancestors) != 12 {
			t.Errorf("Ancestors = %d deep, want 12", len(ancestors))
		}
	})

	t.Run("returns ancestors root first", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		root := create(t, s, tracker.NewItem{})
		mid := child(t, s, root.ID)
		leaf := child(t, s, mid.ID)

		ancestors, err := s.Ancestors(ctx, leaf.ID)
		if err != nil {
			t.Fatalf("Ancestors: %v", err)
		}
		if len(ancestors) != 2 || ancestors[0].ID != root.ID || ancestors[1].ID != mid.ID {
			t.Errorf("Ancestors = %v, want [root mid]", ids(ancestors))
		}
	})

	t.Run("refuses a missing parent", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		_, err := s.CreateItem(ctx, tracker.NewItem{
			Type:   TypeBug,
			Parent: idPtr("ghost"),
			Fields: map[tracker.FieldID]tracker.Value{FieldSummary: tracker.Text("x")},
		})
		if !errors.Is(err, tracker.ErrNotFound) {
			t.Errorf("CreateItem under a missing parent = %v, want ErrNotFound", err)
		}
	})

	t.Run("refuses a cycle", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		root := create(t, s, tracker.NewItem{})
		mid := child(t, s, root.ID)
		leaf := child(t, s, mid.ID)

		// Reparenting the root under its own grandchild closes a loop.
		_, err := s.UpdateItem(ctx, root.ID, root.Version, tracker.Patch{Parent: idPtr(leaf.ID)})
		if !errors.Is(err, tracker.ErrCycle) {
			t.Errorf("reparenting = %v, want ErrCycle", err)
		}
	})

	t.Run("refuses parenting an item to itself", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		_, err := s.UpdateItem(ctx, item.ID, item.Version, tracker.Patch{Parent: idPtr(item.ID)})
		if !errors.Is(err, tracker.ErrCycle) {
			t.Errorf("self-parenting = %v, want ErrCycle", err)
		}
	})

	t.Run("reparents a whole subtree", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		oldParent := create(t, s, tracker.NewItem{})
		newParent := create(t, s, tracker.NewItem{})
		moved := child(t, s, oldParent.ID)
		grandchild := child(t, s, moved.ID)

		if _, err := s.UpdateItem(ctx, moved.ID, moved.Version, tracker.Patch{Parent: idPtr(newParent.ID)}); err != nil {
			t.Fatalf("reparent: %v", err)
		}
		ancestors, err := s.Ancestors(ctx, grandchild.ID)
		if err != nil {
			t.Fatalf("Ancestors: %v", err)
		}
		if len(ancestors) == 0 || ancestors[0].ID != newParent.ID {
			t.Errorf("the subtree did not move: ancestors = %v", ids(ancestors))
		}
	})
}

func ids(items []tracker.Item) []tracker.ItemID {
	out := make([]tracker.ItemID, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}
