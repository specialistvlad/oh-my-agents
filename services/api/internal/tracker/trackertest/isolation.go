package trackertest

import (
	"testing"
	"time"

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
		got.Fields[FieldSummary] = tracker.Text("tampered")
		got.Fields["injected"] = tracker.Text("nope")

		again, err := s.Item(ctx, item.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if summary, _ := again.Fields[FieldSummary].String(); summary == "tampered" {
			t.Error("mutating a returned item reached the store")
		}
		if _, injected := again.Fields["injected"]; injected {
			t.Error("a key added to a returned item reached the store")
		}
	})

	t.Run("does not keep the caller's map", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		fields := map[tracker.FieldID]tracker.Value{FieldSummary: tracker.Text("original")}
		item, err := s.CreateItem(ctx, tracker.NewItem{Type: TypeBug, Fields: fields})
		if err != nil {
			t.Fatalf("CreateItem: %v", err)
		}
		fields[FieldSummary] = tracker.Text("mutated after the write")

		got, err := s.Item(ctx, item.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if summary, _ := got.Fields[FieldSummary].String(); summary != "original" {
			t.Errorf("summary = %q; the store kept the caller's map", summary)
		}
	})

	t.Run("does not hand out its own schema", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		schema, err := s.Schema(ctx)
		if err != nil {
			t.Fatalf("Schema: %v", err)
		}
		typ, ok := schema.Type(TypeBug)
		if !ok || len(typ.Fields) == 0 || len(typ.Statuses) == 0 {
			t.Fatalf("Schema returned an empty type: %+v", typ)
		}
		typ.Fields[0].Name = "tampered"
		typ.Statuses[0].Name = "tampered"

		again, err := s.Schema(ctx)
		if err != nil {
			t.Fatalf("Schema: %v", err)
		}
		fresh, _ := again.Type(TypeBug)
		if fresh.Fields[0].Name == "tampered" || fresh.Statuses[0].Name == "tampered" {
			t.Error("mutating a returned schema reached the store")
		}
	})

	t.Run("does not alias comment pointers", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		posted, err := s.AddComment(ctx, tracker.NewComment{Item: item.ID, Body: "a"})
		if err != nil {
			t.Fatalf("AddComment: %v", err)
		}
		edited, err := s.EditComment(ctx, posted.ID, posted.Version, "b", human("vk"))
		if err != nil {
			t.Fatalf("EditComment: %v", err)
		}
		if edited.EditedAt == nil {
			t.Fatal("EditComment recorded no edit stamp")
		}
		stamp := *edited.EditedAt

		// Every path that hands a comment out has to copy it, so each is
		// mutated in turn and the store re-read after both.
		*edited.EditedAt = stamp.Add(-1000 * time.Hour)
		listed, err := s.Comments(ctx, item.ID, tracker.PageRequest{})
		if err != nil {
			t.Fatalf("Comments: %v", err)
		}
		if !listed.Rows[0].EditedAt.Equal(stamp) {
			t.Error("writing through the comment EditComment returned reached the store")
		}
		*listed.Rows[0].EditedAt = stamp.Add(-2000 * time.Hour)

		again, err := s.Comments(ctx, item.ID, tracker.PageRequest{})
		if err != nil {
			t.Fatalf("Comments: %v", err)
		}
		if !again.Rows[0].EditedAt.Equal(stamp) {
			t.Error("writing through a listed comment reached the store")
		}
	})

	t.Run("does not alias event changes", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if _, err := move(t, s, item, StatusDoing); err != nil {
			t.Fatalf("move: %v", err)
		}
		page, err := s.Events(ctx, tracker.EventQuery{
			Item: &item.ID, Kinds: []tracker.EventKind{tracker.EventStatusChanged},
		})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(page.Rows) != 1 || len(page.Rows[0].Changes) != 1 {
			t.Fatalf("Events = %+v, want one status change", page.Rows)
		}
		*page.Rows[0].Changes[0].To = tracker.Text("rewritten history")

		again, err := s.Events(ctx, tracker.EventQuery{
			Item: &item.ID, Kinds: []tracker.EventKind{tracker.EventStatusChanged},
		})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if to, _ := again.Rows[0].Changes[0].To.String(); to != string(StatusDoing) {
			t.Errorf("recorded history was rewritten through a returned event: %q", to)
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
