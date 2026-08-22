package projectsbus_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects/projectstest"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projectsbus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

func store(t *testing.T) (projects.Store, <-chan bus.Message) {
	t.Helper()
	workspace := t.TempDir()
	records, err := settings.NewFS(filepath.Join(workspace, "shared"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	b := bus.NewMemory()
	t.Cleanup(func() { _ = b.Close() })
	messages, err := b.Subscribe(t.Context())
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	inner := projects.NewRegistry(projects.Deps{Records: records, Workspace: workspace})
	return projectsbus.New(inner, b), messages
}

// Announcing must not change what a store is.
func TestConformance(t *testing.T) {
	projectstest.Run(t, func(t *testing.T) projects.Store {
		s, _ := store(t)
		return s
	})
}

func next(t *testing.T, messages <-chan bus.Message) bus.Message {
	t.Helper()
	select {
	case m := <-messages:
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("no message was published")
		return bus.Message{}
	}
}

func TestAnnouncesTheWholeLifecycle(t *testing.T) {
	s, messages := store(t)
	ctx := t.Context()

	p, err := s.Create(ctx, projects.New{Name: "Watched"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	created := next(t, messages)
	if created.Kind != projectsbus.KindCreated || created.Rooms[0] != projectsbus.Room {
		t.Fatalf("message = %+v, want project.created in the projects room", created)
	}
	// The record travels, so a list needs no fetch after every event.
	var carried projects.Project
	if err := json.Unmarshal(created.Data, &carried); err != nil {
		t.Fatalf("decode %s: %v", created.Data, err)
	}
	if carried.ID != p.ID || carried.Name != "Watched" || carried.Root == "" {
		t.Errorf("payload = %+v, want the whole record", carried)
	}

	if _, err := s.Rename(ctx, p.ID, "Renamed"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if m := next(t, messages); m.Kind != projectsbus.KindChanged {
		t.Errorf("Kind = %q, want project.changed", m.Kind)
	}
	if _, err := s.Repoint(ctx, p.ID, filepath.Join(t.TempDir(), "moved")); err != nil {
		t.Fatalf("Repoint: %v", err)
	}
	if m := next(t, messages); m.Kind != projectsbus.KindChanged {
		t.Errorf("Kind = %q, want project.changed", m.Kind)
	}
	if err := s.Remove(ctx, p.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	removed := next(t, messages)
	if removed.Kind != projectsbus.KindRemoved {
		t.Errorf("Kind = %q, want project.removed", removed.Kind)
	}
	var gone struct {
		ID projects.ID `json:"id"`
	}
	if err := json.Unmarshal(removed.Data, &gone); err != nil || gone.ID != p.ID {
		t.Errorf("removal payload = %s, want the id", removed.Data)
	}
}

// A refused change altered nothing, so announcing it would be a lie.
func TestSaysNothingWhenTheChangeFailed(t *testing.T) {
	s, messages := store(t)
	if _, err := s.Create(t.Context(), projects.New{Name: ""}); err == nil {
		t.Fatal("Create accepted an empty name")
	}
	select {
	case m := <-messages:
		t.Errorf("a failed create was announced: %+v", m)
	case <-time.After(100 * time.Millisecond):
	}
}
