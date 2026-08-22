package fs_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/fs"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/store"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/trackertest"
)

// stepClock advances per reading, so ordering by time is deterministic
// without any test sleeping.
type stepClock struct{ n int }

func (c *stepClock) Now() time.Time {
	c.n++
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(c.n) * time.Second)
}

type countingIDs struct{ n int }

func (g *countingIDs) NewID() string {
	g.n++
	return string(rune('a'+(g.n-1)%26)) + string(rune('0'+(g.n-1)/26))
}

func open(t *testing.T, dir string) tracker.Store {
	t.Helper()
	s, err := fs.New(t.Context(), dir, store.Deps{Clock: &stepClock{}, IDs: &countingIDs{}})
	if err != nil {
		t.Fatalf("fs.New(%q): %v", dir, err)
	}
	return s
}

// The whole point: the same guarantees, now against files.
func TestConformance(t *testing.T) {
	trackertest.Run(t, func(t *testing.T) tracker.Store {
		return open(t, t.TempDir())
	})
}

// A store nobody has written to yet is empty, not an error. A project has a
// tracker from the moment one is asked for.
func TestAnUnusedDirectoryIsAnEmptyTracker(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "never-written")
	s := open(t, dir)

	schema, err := s.Schema(t.Context())
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if len(schema.Types) != 0 {
		t.Errorf("Schema = %+v, want empty", schema)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("opening a tracker created directories; reading must touch nothing")
	}
}

// What durability actually means: close the store, open it again, and
// everything is still there.
func TestStateSurvivesReopening(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()

	before := open(t, dir)
	if err := before.PutItemType(ctx, trackertest.BugType()); err != nil {
		t.Fatalf("PutItemType: %v", err)
	}
	item, err := before.CreateItem(ctx, tracker.NewItem{
		Type: trackertest.TypeBug, Title: "It breaks",
		Fields: map[tracker.FieldID]tracker.Value{trackertest.FieldSummary: tracker.Text("it breaks")},
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if _, err := before.AddComment(ctx, tracker.NewComment{Item: item.ID, Body: "seen"}); err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	after := open(t, dir)
	got, err := after.Item(ctx, item.ID)
	if err != nil {
		t.Fatalf("Item after reopening: %v", err)
	}
	if got.Title != "It breaks" || got.Version != item.Version {
		t.Errorf("Item = %+v, want it unchanged across a reopen", got)
	}
	if summary, _ := got.Fields[trackertest.FieldSummary].String(); summary != "it breaks" {
		t.Errorf("field value did not survive: %q", summary)
	}
	schema, err := after.Schema(ctx)
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if _, ok := schema.Type(trackertest.TypeBug); !ok {
		t.Error("the type did not survive reopening")
	}
	comments, err := after.Comments(ctx, item.ID, tracker.PageRequest{})
	if err != nil {
		t.Fatalf("Comments: %v", err)
	}
	if len(comments.Rows) != 1 {
		t.Errorf("comments = %d, want 1 to have survived", len(comments.Rows))
	}
}

// Sequence numbers must not restart, or a reader that resumed from one would
// re-handle everything after a restart.
func TestEventSequenceContinuesAfterReopening(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()

	before := open(t, dir)
	if err := before.PutItemType(ctx, trackertest.BugType()); err != nil {
		t.Fatalf("PutItemType: %v", err)
	}
	for range 3 {
		if _, err := before.CreateItem(ctx, tracker.NewItem{
			Type: trackertest.TypeBug,
			Fields: map[tracker.FieldID]tracker.Value{
				trackertest.FieldSummary: tracker.Text("x"),
			},
		}); err != nil {
			t.Fatalf("CreateItem: %v", err)
		}
	}
	first, err := before.Events(ctx, tracker.EventQuery{})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	highest := first.Rows[len(first.Rows)-1].Seq

	after := open(t, dir)
	if _, err := after.CreateItem(ctx, tracker.NewItem{
		Type:   trackertest.TypeBug,
		Fields: map[tracker.FieldID]tracker.Value{trackertest.FieldSummary: tracker.Text("y")},
	}); err != nil {
		t.Fatalf("CreateItem after reopening: %v", err)
	}
	next, err := after.Events(ctx, tracker.EventQuery{Since: highest})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(next.Rows) != 1 {
		t.Fatalf("events after %d = %d, want exactly the one new event", highest, len(next.Rows))
	}
	if next.Rows[0].Seq <= highest {
		t.Errorf("sequence restarted: %d is not above %d", next.Rows[0].Seq, highest)
	}
}
