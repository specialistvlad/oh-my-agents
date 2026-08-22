// Package scopes hands out storage already rooted where it belongs.
//
// This is how ADR-0009's scoping is enforced: a caller is given a store that
// is physically incapable of addressing another project, rather than a store
// plus a project argument it must remember to pass. There is no argument to
// get wrong, because isolation is a property of what the caller was handed.
package scopes

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/rooms"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingsbus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Registry is the part of the project store scopes needs: where a project
// lives. Declared here, in the consumer.
type Registry interface {
	Get(ctx context.Context, id projects.ID) (projects.Project, error)
}

// Scopes resolves a project into the stores that belong to it.
type Scopes struct {
	registry Registry
	pub      bus.Publisher

	// One tracker per project, shared. See [Scopes.Tracker] for why that is
	// a correctness requirement rather than a cache.
	mu       sync.Mutex
	trackers map[projects.ID]tracker.Store
}

// New returns a resolver.
func New(registry Registry, pub bus.Publisher) *Scopes {
	return &Scopes{
		registry: registry,
		pub:      pub,
		trackers: make(map[projects.ID]tracker.Store),
	}
}

// Settings returns the settings store for one project, rooted at its own
// directory and announcing to its own room.
//
// A project that does not exist is [projects.ErrNotFound], which is the same
// answer any other project-addressed call gives, so an edge maps one error.
func (s *Scopes) Settings(ctx context.Context, id projects.ID) (settings.Store, error) {
	p, err := s.registry.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	store, err := settings.NewFS(filepath.Join(p.Root, "settings"))
	if err != nil {
		return nil, err
	}
	return settingsbus.New(store, s.pub, rooms.Project(p.ID)), nil
}
