package projects

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

func notFound(id ID) error { return fmt.Errorf("%w: %q", ErrNotFound, id) }

// Rename implements [Store]. Display text only: the id and every path derived
// from it are untouched, which is the reason the two are separate.
func (r *Registry) Rename(ctx context.Context, id ID, name string) (Project, error) {
	if err := ValidateName(name); err != nil {
		return Project{}, err
	}
	p, err := r.Get(ctx, id)
	if err != nil {
		return Project{}, err
	}
	p.Name = strings.TrimSpace(name)
	p.UpdatedAt = r.clock()
	return p, r.put(ctx, p)
}

// Repoint implements [Store]. It moves no files.
//
// Marking the new directory is the loud part: it claims a directory the caller
// may not have thought of as ours, and removal will later delete what it
// claims. Re-pointing is where an accident starts, not removal.
func (r *Registry) Repoint(ctx context.Context, id ID, root string) (Project, error) {
	p, err := r.Get(ctx, id)
	if err != nil {
		return Project{}, err
	}
	if strings.TrimSpace(root) == "" {
		return Project{}, fmt.Errorf("%w: empty", ErrInvalidRoot)
	}
	abs, err := resolveRoot(root, r.workspace)
	if err != nil {
		return Project{}, err
	}
	if err := writeMarker(abs, id); err != nil {
		return Project{}, err
	}
	p.Root = abs
	p.UpdatedAt = r.clock()
	return p, r.put(ctx, p)
}

// Remove implements [Store]: the record and the root directory, wherever it
// lives.
//
// The record goes last. A failure while deleting the directory then leaves the
// project still listed and still removable, rather than an unreachable
// directory nothing remembers — which is the failure the registry being
// authoritative is supposed to prevent.
func (r *Registry) Remove(ctx context.Context, id ID) error {
	p, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	// Re-check the root against the same refusals creation applied. A record
	// edited by hand is exactly the case this is here for.
	abs, err := resolveRoot(p.Root, r.workspace)
	if err != nil {
		return err
	}
	if err := checkMarker(abs, id); err != nil {
		return err
	}
	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("projects: cannot remove %q: %w", abs, err)
	}
	if err := r.records.Delete(ctx, key(id)); err != nil && !errors.Is(err, settings.ErrNotFound) {
		return err
	}
	return nil
}
