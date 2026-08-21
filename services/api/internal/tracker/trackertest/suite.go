// Package trackertest is the conformance suite for [tracker.Store].
//
// ADR-0005 puts enforcement in the adapter: an in-memory store checks the
// rules in Go, a filesystem store will do the same, and a SQL store is
// expected to push them into constraints instead. Rules implemented more than
// once are only replaceable if every implementation agrees, and this suite is
// what says they do. An adapter that has not passed it is not finished.
package trackertest

import (
	"context"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Factory builds a fresh, empty store for one subtest.
type Factory func(t *testing.T) tracker.Store

// Run exercises every guarantee [tracker.Store] documents.
func Run(t *testing.T, newStore Factory) {
	t.Helper()
	groups := map[string]func(*testing.T, Factory){
		"schema":     runSchema,
		"items":      runItems,
		"versioning": runVersioning,
		"workflow":   runWorkflow,
		"hierarchy":  runHierarchy,
		"resolution": runResolution,
		"queries":    runQueries,
		"comments":   runComments,
		"links":      runLinks,
		"events":     runEvents,
		"isolation":  runIsolation,
	}
	for name, group := range groups {
		t.Run(name, func(t *testing.T) { group(t, newStore) })
	}
}

// fixture is a store with the bug type already configured.
func fixture(t *testing.T, newStore Factory) (tracker.Store, context.Context) {
	t.Helper()
	s := newStore(t)
	if err := s.PutItemType(t.Context(), BugType()); err != nil {
		t.Fatalf("PutItemType: %v", err)
	}
	return s, t.Context()
}

// create adds an item and fails the test if it cannot.
func create(t *testing.T, s tracker.Store, n tracker.NewItem) tracker.Item {
	t.Helper()
	if n.Type == "" {
		n.Type = "bug"
	}
	if n.Fields == nil {
		n.Fields = map[tracker.FieldKey]tracker.Value{"summary": tracker.Text("x")}
	}
	item, err := s.CreateItem(t.Context(), n)
	if err != nil {
		t.Fatalf("CreateItem(%+v): %v", n, err)
	}
	return item
}

// child adds an item beneath a parent.
func child(t *testing.T, s tracker.Store, parent tracker.ItemID) tracker.Item {
	t.Helper()
	return create(t, s, tracker.NewItem{Parent: &parent})
}

// move transitions an item and returns the result.
func move(t *testing.T, s tracker.Store, item tracker.Item, to tracker.StatusKey) (tracker.Item, error) {
	t.Helper()
	return s.UpdateItem(t.Context(), item.ID, item.Version, tracker.Patch{Status: &to})
}

// resolve walks an item all the way to fixed, which needs a resolution.
func resolve(t *testing.T, s tracker.Store, item tracker.Item) tracker.Item {
	t.Helper()
	doing, err := move(t, s, item, "doing")
	if err != nil {
		t.Fatalf("open -> doing: %v", err)
	}
	note := tracker.Markdown("done")
	fixed, err := s.UpdateItem(t.Context(), doing.ID, doing.Version, tracker.Patch{
		Status: statusPtr("fixed"),
		Fields: map[tracker.FieldKey]*tracker.Value{"resolution": &note},
	})
	if err != nil {
		t.Fatalf("doing -> fixed: %v", err)
	}
	return fixed
}

func statusPtr(k tracker.StatusKey) *tracker.StatusKey { return &k }

func idPtr(id tracker.ItemID) *tracker.ItemID { return &id }
