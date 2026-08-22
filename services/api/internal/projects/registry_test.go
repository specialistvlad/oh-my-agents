package projects_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects/projectstest"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// counting nonces keep minted ids predictable without making them collide.
type nonces struct{ n int }

func (c *nonces) next() string {
	c.n++
	return string(rune('a'+(c.n-1)%26)) + string(rune('0'+(c.n-1)/26%10)) + "01"
}

func registry(t *testing.T) *projects.Registry {
	t.Helper()
	workspace := t.TempDir()
	records, err := settings.NewFS(filepath.Join(workspace, "shared"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	c := &nonces{}
	return projects.NewRegistry(projects.Deps{
		Records:   records,
		Workspace: workspace,
		Nonce:     c.next,
		Clock:     func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	})
}

func TestConformance(t *testing.T) {
	projectstest.Run(t, func(t *testing.T) projects.Store { return registry(t) })
}

// The default root is inside the workspace, which is what makes removing it
// removing something we made.
func TestDefaultRootIsInTheWorkspace(t *testing.T) {
	r := registry(t)
	p, err := r.Create(t.Context(), projects.New{Name: "Defaulted"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if filepath.Base(filepath.Dir(p.Root)) != "projects" {
		t.Errorf("Root = %q, want it under <workspace>/projects", p.Root)
	}
	if filepath.Base(p.Root) != string(p.ID) {
		t.Errorf("Root = %q, want it named for the id", p.Root)
	}
}

// A root containing the workspace would take the registry with it, so the
// deletion could not even be recorded.
func TestRefusesARootContainingTheWorkspace(t *testing.T) {
	workspace := t.TempDir()
	records, err := settings.NewFS(filepath.Join(workspace, "shared"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	r := projects.NewRegistry(projects.Deps{Records: records, Workspace: workspace})

	_, err = r.Create(t.Context(), projects.New{Name: "Greedy", Root: filepath.Dir(workspace)})
	if !errors.Is(err, projects.ErrInvalidRoot) {
		t.Errorf("Create above the workspace = %v, want ErrInvalidRoot", err)
	}
}

// A project pointed at a directory holding other work takes that work with it.
// This is the documented sharp edge of ADR-0010, pinned so it cannot change
// silently.
func TestRemovingAMarkedRootTakesEverythingInIt(t *testing.T) {
	r := registry(t)
	theirs := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(theirs, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(theirs, "code.go"), []byte("package main"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	p, err := r.Create(t.Context(), projects.New{Name: "Adopted", Root: theirs})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.Remove(t.Context(), p.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(theirs); !os.IsNotExist(err) {
		t.Error("the adopted root survived; ADR-0010 says removal takes it")
	}
}

func TestMintID(t *testing.T) {
	cases := map[string]struct{ name, want string }{
		"words":          {"ACME Website", "acme-website-x"},
		"punctuation":    {"Hello, World!", "hello-world-x"},
		"already slug":   {"acme-site", "acme-site-x"},
		"nothing usable": {"日本語", "project-x"},
		"very long": {
			"a very long project name that keeps going well past any reasonable limit",
			"a-very-long-project-name-that-keeps-goin-x",
		},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			got := projects.MintID(c.name, "x")
			if string(got) != c.want {
				t.Errorf("MintID(%q) = %q, want %q", c.name, got, c.want)
			}
			if err := got.Validate(); err != nil {
				t.Errorf("minted %q is not a valid id: %v", got, err)
			}
		})
	}
}

// The stem is frozen: renaming leaves it stale, and that is the point.
func TestRenamingLeavesTheIDStale(t *testing.T) {
	r := registry(t)
	p, err := r.Create(t.Context(), projects.New{Name: "Original Name"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	renamed, err := r.Rename(t.Context(), p.ID, "Something Else Entirely")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if renamed.ID != p.ID {
		t.Fatalf("ID changed on rename: %q -> %q", p.ID, renamed.ID)
	}
	if got := string(renamed.ID); got[:len("original-name")] != "original-name" {
		t.Errorf("id = %q; the stem should stay as minted", got)
	}
}
