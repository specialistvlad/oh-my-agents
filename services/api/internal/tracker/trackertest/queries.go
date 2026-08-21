package trackertest

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func runQueries(t *testing.T, newStore Factory) {
	t.Run("finds everything by default", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		for range 3 {
			create(t, s, tracker.NewItem{})
		}
		page, err := s.FindItems(ctx, tracker.Query{})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(page.Rows) != 3 {
			t.Errorf("found %d, want 3", len(page.Rows))
		}
	})

	// Asking by category is what lets logic say "everything unfinished"
	// without naming a single user-defined status.
	t.Run("filters by category", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		create(t, s, tracker.NewItem{})
		resolve(t, s, create(t, s, tracker.NewItem{}))

		open, err := s.FindItems(ctx, tracker.Query{
			Categories: []tracker.StatusCategory{tracker.CategoryBacklog},
		})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(open.Rows) != 1 {
			t.Errorf("backlog = %d items, want 1", len(open.Rows))
		}
		done, err := s.FindItems(ctx, tracker.Query{
			Categories: []tracker.StatusCategory{tracker.CategoryDone},
		})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(done.Rows) != 1 {
			t.Errorf("done = %d items, want 1", len(done.Rows))
		}
	})

	t.Run("filters by status and by field", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		create(t, s, tracker.NewItem{Fields: map[tracker.FieldKey]tracker.Value{
			"summary": tracker.Text("x"), "severity": tracker.Select("high"),
		}})
		create(t, s, tracker.NewItem{})

		high, err := s.FindItems(ctx, tracker.Query{
			Statuses: []tracker.StatusKey{"open"},
			Fields:   []tracker.FieldMatch{{Field: "severity", Equals: tracker.Select("high")}},
		})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(high.Rows) != 1 {
			t.Errorf("high severity = %d items, want 1", len(high.Rows))
		}
	})

	t.Run("filters by parent, roots and subtree", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		root := create(t, s, tracker.NewItem{})
		mid := child(t, s, root.ID)
		child(t, s, mid.ID)

		children, err := s.FindItems(ctx, tracker.Query{Parent: idPtr(root.ID)})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(children.Rows) != 1 {
			t.Errorf("direct children = %d, want 1", len(children.Rows))
		}

		roots, err := s.FindItems(ctx, tracker.Query{Roots: true})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(roots.Rows) != 1 || roots.Rows[0].ID != root.ID {
			t.Errorf("roots = %v, want just the root", ids(roots.Rows))
		}

		// Subtree reaches all the way down and excludes the root itself.
		below, err := s.FindItems(ctx, tracker.Query{Subtree: idPtr(root.ID)})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(below.Rows) != 2 {
			t.Errorf("subtree = %d items, want 2", len(below.Rows))
		}
	})

	t.Run("sorts and pages", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		for _, title := range []string{"c", "a", "b"} {
			create(t, s, tracker.NewItem{Title: title})
		}
		first, err := s.FindItems(ctx, tracker.Query{
			Sort: tracker.Sort{By: tracker.SortTitle},
			Page: tracker.PageRequest{Limit: 2},
		})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(first.Rows) != 2 || first.Rows[0].Title != "a" || first.Rows[1].Title != "b" {
			t.Fatalf("first page = %v, want [a b]", titles(first.Rows))
		}
		if first.Next == "" {
			t.Fatal("first page carries no cursor, but a third item exists")
		}
		second, err := s.FindItems(ctx, tracker.Query{
			Sort: tracker.Sort{By: tracker.SortTitle},
			Page: tracker.PageRequest{Limit: 2, Cursor: first.Next},
		})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(second.Rows) != 1 || second.Rows[0].Title != "c" {
			t.Errorf("second page = %v, want [c]", titles(second.Rows))
		}
		if second.Next != "" {
			t.Error("the last page should carry no cursor")
		}
	})

	t.Run("sorts descending", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		for _, title := range []string{"a", "b"} {
			create(t, s, tracker.NewItem{Title: title})
		}
		page, err := s.FindItems(ctx, tracker.Query{
			Sort: tracker.Sort{By: tracker.SortTitle, Desc: true},
		})
		if err != nil {
			t.Fatalf("FindItems: %v", err)
		}
		if len(page.Rows) != 2 || page.Rows[0].Title != "b" {
			t.Errorf("descending = %v, want [b a]", titles(page.Rows))
		}
	})

	t.Run("rejects a cursor it did not issue", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		_, err := s.FindItems(ctx, tracker.Query{Page: tracker.PageRequest{Cursor: "nonsense"}})
		if !errors.Is(err, tracker.ErrInvalidCursor) {
			t.Errorf("FindItems = %v, want ErrInvalidCursor", err)
		}
	})
}

func titles(items []tracker.Item) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Title
	}
	return out
}
