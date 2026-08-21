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
		return settings.NewFS(t.TempDir())
	})
}

func TestFSDefaultsToDotOma(t *testing.T) {
	if got := settings.NewFS("").Dir(); got != settings.DefaultDir {
		t.Errorf("Dir = %q, want %q", got, settings.DefaultDir)
	}
	if settings.DefaultDir != ".oma" {
		t.Errorf("DefaultDir = %q, want .oma", settings.DefaultDir)
	}
}

func TestFSCreatesNothingUntilWritten(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "root")
	s := settings.NewFS(dir)

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
	s := settings.NewFS(dir)
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
	s := settings.NewFS(dir)
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
