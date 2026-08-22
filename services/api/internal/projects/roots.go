package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveRoot turns a requested root into an absolute path and refuses the
// ones a mistake must not be able to reach.
//
// These refusals exist because removal deletes the root. A path where a
// mistake is unrecoverable rather than merely annoying is one this system
// should decline to be pointed at, whatever the caller intended.
func resolveRoot(root, workspace string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("%w: %q: %w", ErrInvalidRoot, root, err)
	}
	abs = filepath.Clean(abs)

	if abs == string(filepath.Separator) {
		return "", fmt.Errorf("%w: the filesystem root", ErrInvalidRoot)
	}
	if home, err := os.UserHomeDir(); err == nil && abs == filepath.Clean(home) {
		return "", fmt.Errorf("%w: a home directory", ErrInvalidRoot)
	}
	// An ancestor of the workspace would take the workspace with it, and the
	// registry recording the deletion along with it.
	if workspace != "" && isAncestor(abs, filepath.Clean(workspace)) {
		return "", fmt.Errorf("%w: %q contains the workspace", ErrInvalidRoot, abs)
	}
	return abs, nil
}

// isAncestor reports whether parent contains child, or is child.
func isAncestor(parent, child string) bool {
	if parent == child {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
