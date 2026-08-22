package projects

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// MarkerName is the file that says a directory is a project's root.
//
// It is what makes "remove deletes the root" safe to mean. Removal reads it
// and refuses anything it does not recognize, so the destructive path is
// bounded to directories this system was told are projects — not to whatever
// a record happens to contain after a bad edit.
//
// The name is distinctive on purpose: writing it into a directory a user chose
// must not overwrite something of theirs.
const MarkerName = ".oma-project.json"

type marker struct {
	ID ID `json:"id"`
}

// writeMarker claims a directory for a project, creating it if needed.
func writeMarker(root string, id ID) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("%w: cannot create %q: %w", ErrInvalidRoot, root, err)
	}
	body, err := json.Marshal(marker{ID: id})
	if err != nil {
		return fmt.Errorf("projects: encode marker: %w", err)
	}
	path := filepath.Join(root, MarkerName)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("%w: cannot mark %q: %w", ErrInvalidRoot, root, err)
	}
	return nil
}

// checkMarker reports whether root is this project's root and nobody else's.
// A missing directory is not an error here: the record outliving its directory
// is a state removal should be able to tidy up.
func checkMarker(root string, id ID) error {
	body, err := os.ReadFile(filepath.Join(root, MarkerName)) //nolint:gosec // path is an absolute recorded root
	if errors.Is(err, fs.ErrNotExist) {
		if _, statErr := os.Stat(root); errors.Is(statErr, fs.ErrNotExist) {
			return nil // nothing on disk to protect
		}
		return fmt.Errorf("%w: %q holds no %s", ErrNotAProjectRoot, root, MarkerName)
	}
	if err != nil {
		return fmt.Errorf("%w: cannot read the marker in %q: %w", ErrNotAProjectRoot, root, err)
	}
	var held marker
	if err := json.Unmarshal(body, &held); err != nil {
		return fmt.Errorf("%w: the marker in %q is unreadable", ErrNotAProjectRoot, root)
	}
	if held.ID != id {
		return fmt.Errorf("%w: %q belongs to %q, not %q", ErrNotAProjectRoot, root, held.ID, id)
	}
	return nil
}
