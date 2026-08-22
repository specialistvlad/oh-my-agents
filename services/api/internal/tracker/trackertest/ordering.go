package trackertest

import (
	"context"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Ordering is what a board is built on, so it is held to the same standard as
// the rules: every adapter must produce the same order from the same moves
// (ADR-0005, ADR-0013).
func runOrdering(t *testing.T, newStore Factory) {
	t.Run("new items append", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		var made []tracker.ItemID
		for _, title := range []string{"first", "second", "third"} {
			made = append(made, create(t, s, tracker.NewItem{Title: title}).ID)
		}
		if got := orderOf(t, s, ctx); !sameOrder(got, made) {
			t.Errorf("order = %v, want them in the order they were created", got)
		}
	})

	t.Run("moves to the front", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a, b, c := three(t, s)

		if err := s.Reorder(ctx, c, nil, &a); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		if got := orderOf(t, s, ctx); !sameOrder(got, []tracker.ItemID{c, a, b}) {
			t.Errorf("order = %v, want c first", got)
		}
	})

	t.Run("moves to the end", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a, b, c := three(t, s)

		if err := s.Reorder(ctx, a, &c, nil); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		if got := orderOf(t, s, ctx); !sameOrder(got, []tracker.ItemID{b, c, a}) {
			t.Errorf("order = %v, want a last", got)
		}
	})

	t.Run("moves between two neighbors", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a, b, c := three(t, s)

		if err := s.Reorder(ctx, c, &a, &b); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		if got := orderOf(t, s, ctx); !sameOrder(got, []tracker.ItemID{a, c, b}) {
			t.Errorf("order = %v, want c between a and b", got)
		}
	})

	// Squeezing repeatedly into the same gap must keep working. A sparse key
	// is the whole reason this does not renumber siblings (ADR-0013).
	t.Run("survives repeated insertion at one spot", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a, b, _ := three(t, s)
		for range 30 {
			moved := create(t, s, tracker.NewItem{Title: "squeezed"}).ID
			if err := s.Reorder(ctx, moved, &a, &b); err != nil {
				t.Fatalf("Reorder: %v", err)
			}
			b = moved
		}
		order := orderOf(t, s, ctx)
		if order[0] != a {
			t.Errorf("order starts %v, want a first", order[0])
		}
	})

	// A drag is not an edit (ADR-0013). If it bumped the version, dragging a
	// card would conflict with somebody editing its description.
	t.Run("does not bump the version", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a, _, c := three(t, s)

		before, err := s.Item(ctx, a)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if err := s.Reorder(ctx, a, &c, nil); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		after, err := s.Item(ctx, a)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if after.Version != before.Version {
			t.Errorf("Version %d -> %d; a drag must not bump it", before.Version, after.Version)
		}
		// And the edit that was already in flight still applies.
		title := "edited after the drag"
		if _, err := s.UpdateItem(ctx, a, before.Version, tracker.Patch{Title: &title}); err != nil {
			t.Errorf("an edit at the pre-drag version was refused: %v", err)
		}
	})

	// The reverse of the same rule: an edit must not revert a drag.
	t.Run("an edit does not revert a move", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a, _, c := three(t, s)

		stale, err := s.Item(ctx, a)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if err := s.Reorder(ctx, a, &c, nil); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		title := "edited"
		if _, err := s.UpdateItem(ctx, a, stale.Version, tracker.Patch{Title: &title}); err != nil {
			t.Fatalf("UpdateItem: %v", err)
		}
		if got := orderOf(t, s, ctx); got[len(got)-1] != a {
			t.Errorf("order = %v; the edit reverted the move", got)
		}
	})

	t.Run("refuses a neighbor that does not exist", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a, _, _ := three(t, s)
		ghost := tracker.ItemID("ghost")
		if err := s.Reorder(ctx, a, &ghost, nil); err == nil {
			t.Error("Reorder accepted a neighbor that does not exist")
		}
		if err := s.Reorder(ctx, "ghost", nil, nil); err == nil {
			t.Error("Reorder accepted an item that does not exist")
		}
	})
}

// three creates three items and returns them in order.
func three(t *testing.T, s tracker.Store) (a, b, c tracker.ItemID) {
	t.Helper()
	return create(t, s, tracker.NewItem{Title: "a"}).ID,
		create(t, s, tracker.NewItem{Title: "b"}).ID,
		create(t, s, tracker.NewItem{Title: "c"}).ID
}

func orderOf(t *testing.T, s tracker.Store, ctx context.Context) []tracker.ItemID {
	t.Helper()
	page, err := s.FindItems(ctx, tracker.Query{Sort: tracker.Sort{By: tracker.SortRank}})
	if err != nil {
		t.Fatalf("FindItems: %v", err)
	}
	return ids(page.Rows)
}

func sameOrder(got, want []tracker.ItemID) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
