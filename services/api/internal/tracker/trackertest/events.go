package trackertest

import (
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func runEvents(t *testing.T, newStore Factory) {
	t.Run("records creation", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})

		page, err := s.Events(ctx, tracker.EventQuery{Item: &item.ID})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(page.Rows) != 1 || page.Rows[0].Kind != tracker.EventItemCreated {
			t.Errorf("Events = %v, want one item_created", kinds(page.Rows))
		}
	})

	// Status moves get their own kind: reacting to "this was closed" should
	// not mean sifting a generic update for a reserved key.
	t.Run("calls out status changes", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if _, err := move(t, s, item, "doing"); err != nil {
			t.Fatalf("move: %v", err)
		}
		page, err := s.Events(ctx, tracker.EventQuery{
			Item: &item.ID, Kinds: []tracker.EventKind{tracker.EventStatusChanged},
		})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(page.Rows) != 1 {
			t.Fatalf("status events = %d, want 1", len(page.Rows))
		}
		changes := page.Rows[0].Changes
		if len(changes) != 1 || changes[0].Field != tracker.FieldStatus {
			t.Fatalf("Changes = %+v, want one @status change", changes)
		}
		from, _ := changes[0].From.String()
		to, _ := changes[0].To.String()
		if from != "open" || to != "doing" {
			t.Errorf("change = %q -> %q, want open -> doing", from, to)
		}
	})

	t.Run("records field edits with before and after", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		high := tracker.Select("high")
		if _, err := s.UpdateItem(ctx, item.ID, item.Version, tracker.Patch{
			Fields: map[tracker.FieldKey]*tracker.Value{"severity": &high},
		}); err != nil {
			t.Fatalf("UpdateItem: %v", err)
		}
		page, err := s.Events(ctx, tracker.EventQuery{
			Item: &item.ID, Kinds: []tracker.EventKind{tracker.EventItemUpdated},
		})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(page.Rows) != 1 || len(page.Rows[0].Changes) != 1 {
			t.Fatalf("Events = %+v, want one update carrying one change", page.Rows)
		}
		change := page.Rows[0].Changes[0]
		was, _ := change.From.Select()
		now, _ := change.To.Select()
		if was != "medium" || now != "high" {
			t.Errorf("change = %q -> %q, want medium -> high", was, now)
		}
	})

	t.Run("records comments and links", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if _, err := s.AddComment(ctx, tracker.NewComment{Item: item.ID, Body: "x"}); err != nil {
			t.Fatalf("AddComment: %v", err)
		}
		page, err := s.Events(ctx, tracker.EventQuery{
			Item: &item.ID, Kinds: []tracker.EventKind{tracker.EventCommentAdded},
		})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(page.Rows) != 1 {
			t.Errorf("comment events = %d, want 1", len(page.Rows))
		}
	})

	// Resuming from the last sequence handled is the whole point of the
	// feed: a reader must never re-handle what it has already seen.
	t.Run("resumes from a sequence", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if _, err := move(t, s, item, "doing"); err != nil {
			t.Fatalf("move: %v", err)
		}
		all, err := s.Events(ctx, tracker.EventQuery{Item: &item.ID})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(all.Rows) < 2 {
			t.Fatalf("Events = %d, want at least 2", len(all.Rows))
		}
		for i := 1; i < len(all.Rows); i++ {
			if all.Rows[i].Seq <= all.Rows[i-1].Seq {
				t.Fatalf("sequences are not increasing: %d then %d", all.Rows[i-1].Seq, all.Rows[i].Seq)
			}
		}
		rest, err := s.Events(ctx, tracker.EventQuery{Item: &item.ID, Since: all.Rows[0].Seq})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if got, want := len(rest.Rows), len(all.Rows)-1; got != want {
			t.Errorf("resumed = %d events, want %d", got, want)
		}
	})
}

func kinds(events []tracker.Event) []tracker.EventKind {
	out := make([]tracker.EventKind, len(events))
	for i, e := range events {
		out[i] = e.Kind
	}
	return out
}
