package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings/settingstest"
)

func TestFSConformance(t *testing.T) {
	settingstest.Run(t, func(t *testing.T) settings.Store {
		s, err := settings.NewFS(t.TempDir())
		if err != nil {
			t.Fatalf("NewFS: %v", err)
		}
		return s
	})
}

func TestFSDefaultsToDotOmaInTheUserHome(t *testing.T) {
	if settings.DirName != ".oma" {
		t.Errorf("DirName = %q, want .oma", settings.DirName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory on this host: %v", err)
	}
	want := filepath.Join(home, ".oma")

	def, err := settings.DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir: %v", err)
	}
	if def != want {
		t.Errorf("DefaultDir = %q, want %q", def, want)
	}

	s, err := settings.NewFS("")
	if err != nil {
		t.Fatalf("NewFS(\"\"): %v", err)
	}
	if s.Dir() != want {
		t.Errorf("NewFS(\"\").Dir = %q, want %q", s.Dir(), want)
	}
}

func TestFSExpandsTildeAndRelativePaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory on this host: %v", err)
	}
	// A container hands OMA_HOME over with no shell to expand it.
	s, err := settings.NewFS("~/workspace")
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	if want := filepath.Join(home, "workspace"); s.Dir() != want {
		t.Errorf("Dir = %q, want %q", s.Dir(), want)
	}

	// A relative override is bound to the startup directory, absolutely, so
	// a later chdir cannot move the workspace out from under the process.
	rel, err := settings.NewFS("./local-oma")
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	if !filepath.IsAbs(rel.Dir()) {
		t.Errorf("Dir = %q, want an absolute path", rel.Dir())
	}
}

func TestFSCreatesNothingUntilWritten(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "root")
	s, err := settings.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	if _, err := s.Keys(t.Context()); err != nil {
		t.Fatalf("Keys: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("reading created %q; construction and reads must not touch disk", dir)
	}

	if err := s.Set(t.Context(), "k", settings.Document(`{}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("Set did not create the root: %v", err)
	}
}

func TestFSStoresOneReadableFilePerKey(t *testing.T) {
	dir := t.TempDir()
	s, err := settings.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	if err := s.Set(t.Context(), "agent/model", settings.Document(`{"m":"opus"}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// The on-disk layout is part of the deal: a human can read and edit
	// these files without the application running.
	path := filepath.Join(dir, "settings", "agent", "model.json")
	got, err := os.ReadFile(path) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("expected a document at %s: %v", path, err)
	}
	if string(got) != `{"m":"opus"}` {
		t.Errorf("file = %s, want the document as written", got)
	}
}

func TestFSLeavesNoTempFilesBehind(t *testing.T) {
	dir := t.TempDir()
	s, err := settings.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	for range 3 {
		if err := s.Set(t.Context(), "k", settings.Document(`{"v":1}`)); err != nil {
			t.Fatalf("Set: %v", err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, "settings"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("found %d entries, want 1: atomic writes leaked temp files", len(entries))
	}
}
