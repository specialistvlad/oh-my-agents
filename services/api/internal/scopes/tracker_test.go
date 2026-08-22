package scopes_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/scopes"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func resolver(t *testing.T) (*scopes.Scopes, projects.Project) {
	t.Helper()
	workspace := t.TempDir()
	records, err := settings.NewFS(filepath.Join(workspace, "shared"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	registry := projects.NewRegistry(projects.Deps{Records: records, Workspace: workspace})
	p, err := registry.Create(t.Context(), projects.New{Name: "Scoped"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	b := bus.NewMemory()
	t.Cleanup(func() { _ = b.Close() })
	return scopes.New(registry, b), p
}

// A tracker holds its state in memory and writes through, so two instances
// over one directory would each believe they had the whole story. Sharing one
// is a correctness requirement, not a cache.
func TestOneTrackerPerProject(t *testing.T) {
	s, p := resolver(t)
	first, err := s.Tracker(t.Context(), p.ID)
	if err != nil {
		t.Fatalf("Tracker: %v", err)
	}
	second, err := s.Tracker(t.Context(), p.ID)
	if err != nil {
		t.Fatalf("Tracker: %v", err)
	}
	if first != second {
		t.Fatal("two calls returned different trackers for one project")
	}

	// A write through one is visible through the other, which is the thing
	// two instances would get wrong.
	item, err := first.CreateItem(t.Context(), tracker.NewItem{Type: scopes.StarterType, Title: "x"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if _, err := second.Item(t.Context(), item.ID); err != nil {
		t.Errorf("the second tracker cannot see the first's write: %v", err)
	}
}

// Without a type a project has a tracker that can hold nothing, and nothing
// yet authors one.
func TestANewTrackerHasAStarterType(t *testing.T) {
	s, p := resolver(t)
	tr, err := s.Tracker(t.Context(), p.ID)
	if err != nil {
		t.Fatalf("Tracker: %v", err)
	}
	schema, err := tr.Schema(t.Context())
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if _, ok := schema.Type(scopes.StarterType); !ok {
		t.Fatalf("schema = %+v, want the starter type", schema)
	}
	if _, err := tr.CreateItem(t.Context(), tracker.NewItem{Type: scopes.StarterType, Title: "usable"}); err != nil {
		t.Errorf("a brand new tracker cannot hold an item: %v", err)
	}
}

// Seeding must never come back after the type is deleted, or a project could
// not get rid of it.
func TestTheStarterTypeIsNotReseeded(t *testing.T) {
	s, p := resolver(t)
	tr, err := s.Tracker(t.Context(), p.ID)
	if err != nil {
		t.Fatalf("Tracker: %v", err)
	}
	replacement := scopes.StarterTaskType()
	replacement.ID = "story-0002"
	replacement.Name = "Story"
	if err := tr.PutItemType(t.Context(), replacement); err != nil {
		t.Fatalf("PutItemType: %v", err)
	}
	if err := tr.DeleteItemType(t.Context(), scopes.StarterType); err != nil {
		t.Fatalf("DeleteItemType: %v", err)
	}

	// A fresh resolver over the same directory, as a restart would give.
	again, err := scopes.New(&onlyProject{p: p}, nil).Tracker(t.Context(), p.ID)
	if err != nil {
		t.Fatalf("Tracker: %v", err)
	}
	schema, err := again.Schema(t.Context())
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if _, ok := schema.Type(scopes.StarterType); ok {
		t.Error("the starter type came back after being deleted")
	}
}

// onlyProject is a registry holding one project, for reopening its tracker.
type onlyProject struct{ p projects.Project }

func (o *onlyProject) Get(_ context.Context, id projects.ID) (projects.Project, error) {
	if id != o.p.ID {
		return projects.Project{}, projects.ErrNotFound
	}
	return o.p, nil
}
