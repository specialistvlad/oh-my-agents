// Package projectstest is the conformance suite for [projects.Store].
//
// Removal deletes a directory, so the guarantees here are not only about
// behavior but about blast radius. Every refusal that keeps a mistake from
// becoming a recursive delete is asserted, because those are exactly the paths
// nobody exercises by hand.
package projectstest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
)

// Factory builds a fresh, empty store for one subtest.
type Factory func(t *testing.T) projects.Store

// Run exercises every guarantee [projects.Store] documents.
func Run(t *testing.T, newStore Factory) {
	t.Helper()
	for name, check := range map[string]func(*testing.T, Factory){
		"creates":           testCreate,
		"lists in order":    testList,
		"renames":           testRename,
		"repoints":          testRepoint,
		"removes":           testRemove,
		"refuses bad input": testRefusesBadInput,
		"guards removal":    testGuardsRemoval,
	} {
		t.Run(name, func(t *testing.T) { check(t, newStore) })
	}
}

func create(t *testing.T, s projects.Store, name string) projects.Project {
	t.Helper()
	p, err := s.Create(t.Context(), projects.New{Name: name})
	if err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}
	return p
}

func testCreate(t *testing.T, newStore Factory) {
	s := newStore(t)
	p := create(t, s, "ACME Website")

	if err := p.ID.Validate(); err != nil {
		t.Errorf("minted id %q is not valid: %v", p.ID, err)
	}
	// A readable stem is the whole reason ids are not opaque.
	if got := string(p.ID); got[:len("acme-website")] != "acme-website" {
		t.Errorf("id = %q, want it to start with a stem from the name", got)
	}
	if p.Name != "ACME Website" {
		t.Errorf("Name = %q, want it kept as typed", p.Name)
	}
	if !filepath.IsAbs(p.Root) {
		t.Errorf("Root = %q, want an absolute path", p.Root)
	}
	if _, err := os.Stat(filepath.Join(p.Root, projects.MarkerName)); err != nil {
		t.Errorf("no marker in the new root: %v", err)
	}
	got, err := s.Get(t.Context(), p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != p.ID || got.Root != p.Root {
		t.Errorf("Get = %+v, want the project as created", got)
	}
}

// Two projects with the same name must not collide, which is what the nonce
// half of an id is for.
func testList(t *testing.T, newStore Factory) {
	s := newStore(t)
	create(t, s, "zebra")
	create(t, s, "apple")
	first := create(t, s, "apple")

	all, err := s.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("List = %d projects, want 3", len(all))
	}
	if all[0].Name != "apple" || all[2].Name != "zebra" {
		t.Errorf("List order = %v, want sorted by name", names(all))
	}
	if all[0].ID == all[1].ID {
		t.Errorf("two projects named the same share an id: %q", first.ID)
	}
}

// A rename must not disturb the id or the root, or every path and room
// address derived from them would move with it.
func testRename(t *testing.T, newStore Factory) {
	s := newStore(t)
	p := create(t, s, "Before")

	renamed, err := s.Rename(t.Context(), p.ID, "After")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if renamed.Name != "After" {
		t.Errorf("Name = %q, want After", renamed.Name)
	}
	if renamed.ID != p.ID || renamed.Root != p.Root {
		t.Errorf("rename moved the id or root: %+v", renamed)
	}
}

// Re-pointing moves no files. It marks the new directory, which is what makes
// removal willing to delete it later.
func testRepoint(t *testing.T, newStore Factory) {
	s := newStore(t)
	p := create(t, s, "Movable")
	if err := os.WriteFile(filepath.Join(p.Root, "keepsake"), []byte("x"), 0o600); err != nil {
		t.Fatalf("seeding the old root: %v", err)
	}
	elsewhere := filepath.Join(t.TempDir(), "elsewhere")

	moved, err := s.Repoint(t.Context(), p.ID, elsewhere)
	if err != nil {
		t.Fatalf("Repoint: %v", err)
	}
	if moved.Root != elsewhere {
		t.Errorf("Root = %q, want %q", moved.Root, elsewhere)
	}
	if _, err := os.Stat(filepath.Join(elsewhere, projects.MarkerName)); err != nil {
		t.Errorf("the new root was not marked: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.Root, "keepsake")); err != nil {
		t.Errorf("Repoint moved files; it must only re-point: %v", err)
	}
}

func testRemove(t *testing.T, newStore Factory) {
	s := newStore(t)
	p := create(t, s, "Doomed")

	if err := s.Remove(t.Context(), p.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := s.Get(t.Context(), p.ID); !errors.Is(err, projects.ErrNotFound) {
		t.Errorf("Get after Remove = %v, want ErrNotFound", err)
	}
	if _, err := os.Stat(p.Root); !os.IsNotExist(err) {
		t.Errorf("the root survived removal: %v", err)
	}
	if err := s.Remove(t.Context(), p.ID); !errors.Is(err, projects.ErrNotFound) {
		t.Errorf("second Remove = %v, want ErrNotFound", err)
	}
}

func testRefusesBadInput(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()

	for name, in := range map[string]projects.New{
		"empty name":    {Name: ""},
		"blank name":    {Name: "   "},
		"enormous name": {Name: string(make([]byte, 500))},
		"root at /":     {Name: "ok", Root: "/"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := s.Create(ctx, in); err == nil {
				t.Errorf("Create(%+v) was accepted", in)
			}
		})
	}
	if _, err := s.Get(ctx, "Not A Valid Id"); !errors.Is(err, projects.ErrInvalidID) {
		t.Errorf("Get with a malformed id = %v, want ErrInvalidID", err)
	}
	if _, err := s.Get(ctx, "absent-0000"); !errors.Is(err, projects.ErrNotFound) {
		t.Errorf("Get of an absent project = %v, want ErrNotFound", err)
	}
}

// The point of the marker: removal must refuse a directory nobody told it was
// a project, however the record came to point there.
func testGuardsRemoval(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()
	p := create(t, s, "Guarded")

	// Someone else's directory, pointed at without going through Repoint.
	theirs := filepath.Join(t.TempDir(), "their-repo")
	if err := os.MkdirAll(theirs, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(theirs, "important.txt"), []byte("work"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Remove(filepath.Join(p.Root, projects.MarkerName)); err != nil {
		t.Fatalf("removing the marker: %v", err)
	}
	if err := s.Remove(ctx, p.ID); !errors.Is(err, projects.ErrNotAProjectRoot) {
		t.Errorf("Remove of an unmarked root = %v, want ErrNotAProjectRoot", err)
	}
	if _, err := os.Stat(p.Root); err != nil {
		t.Errorf("a refused removal deleted the directory anyway: %v", err)
	}

	// A marker naming somebody else is refused too.
	if err := os.WriteFile(
		filepath.Join(p.Root, projects.MarkerName), []byte(`{"id":"someone-else-0001"}`), 0o600,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := s.Remove(ctx, p.ID); !errors.Is(err, projects.ErrNotAProjectRoot) {
		t.Errorf("Remove of another project's root = %v, want ErrNotAProjectRoot", err)
	}
}

func names(all []projects.Project) []string {
	out := make([]string, len(all))
	for i, p := range all {
		out[i] = p.Name
	}
	return out
}
