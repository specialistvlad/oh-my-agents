package trackertest

import (
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// A store must hand out copies. A caller that modifies what it was given
// cannot be allowed to reach inside the store, or every guarantee above it
// becomes advisory.
func runIsolation(t *testing.T, newStore Factory) {
	t.Run("does not hand out its own maps", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})

		got, err := s.Item(ctx, item.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		got.Fields["summary"] = tracker.Text("tampered")
		got.Fields["injected"] = tracker.Text("nope")

		again, err := s.Item(ctx, item.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if summary, _ := again.Fields["summary"].String(); summary == "tampered" {
			t.Error("mutating a returned item reached the store")
		}
		if _, injected := again.Fields["injected"]; injected {
			t.Error("a key added to a returned item reached the store")
		}
	})

	t.Run("does not keep the caller's map", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		fields := map[tracker.FieldKey]tracker.Value{"summary": tracker.Text("original")}
		item, err := s.CreateItem(ctx, tracker.NewItem{Type: "bug", Fields: fields})
		if err != nil {
			t.Fatalf("CreateItem: %v", err)
		}
		fields["summary"] = tracker.Text("mutated after the write")

		got, err := s.Item(ctx, item.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if summary, _ := got.Fields["summary"].String(); summary != "original" {
			t.Errorf("summary = %q; the store kept the caller's map", summary)
		}
	})

	t.Run("does not alias the parent pointer", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		parent := create(t, s, tracker.NewItem{})
		kid := child(t, s, parent.ID)

		got, err := s.Item(ctx, kid.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		*got.Parent = "tampered"

		again, err := s.Item(ctx, kid.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if *again.Parent != parent.ID {
			t.Errorf("Parent = %q; writing through the returned pointer reached the store", *again.Parent)
		}
	})
}
