// Package projectsbus announces project lifecycle changes on a bus.
//
// It wraps a [projects.Store] and implements the same interface, so callers
// cannot tell and nothing above it changes. Announcing at the store rather
// than at an edge is what keeps HTTP and the socket equally audible: an edge
// that announces its own writes stays silent about everybody else's.
package projectsbus

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
)

// Room is where project activity is published.
//
// It sits in the shared scope, above any project, because a client watching
// the list is not yet watching a project — and one that just lost its project
// still needs to hear that it is gone (ADR-0010).
const Room bus.Room = "projects"

// Event kinds.
const (
	KindCreated = "project.created"
	KindChanged = "project.changed"
	KindRemoved = "project.removed"
)

// Store is a [projects.Store] that announces what it changes.
type Store struct {
	inner projects.Store
	pub   bus.Publisher
}

// New wraps a store. A nil publisher announces nothing, so a process with no
// realtime surface wires the same way.
func New(inner projects.Store, pub bus.Publisher) *Store {
	return &Store{inner: inner, pub: pub}
}

// Get implements [projects.Store].
func (s *Store) Get(ctx context.Context, id projects.ID) (projects.Project, error) {
	return s.inner.Get(ctx, id)
}

// List implements [projects.Store].
func (s *Store) List(ctx context.Context) ([]projects.Project, error) {
	return s.inner.List(ctx)
}

// Create implements [projects.Store].
func (s *Store) Create(ctx context.Context, n projects.New) (projects.Project, error) {
	p, err := s.inner.Create(ctx, n)
	return s.announceProject(ctx, KindCreated, p, err)
}

// Rename implements [projects.Store].
func (s *Store) Rename(ctx context.Context, id projects.ID, name string) (projects.Project, error) {
	p, err := s.inner.Rename(ctx, id, name)
	return s.announceProject(ctx, KindChanged, p, err)
}

// Repoint implements [projects.Store].
func (s *Store) Repoint(ctx context.Context, id projects.ID, root string) (projects.Project, error) {
	p, err := s.inner.Repoint(ctx, id, root)
	return s.announceProject(ctx, KindChanged, p, err)
}

// Remove implements [projects.Store].
func (s *Store) Remove(ctx context.Context, id projects.ID) error {
	if err := s.inner.Remove(ctx, id); err != nil {
		return err
	}
	s.publish(ctx, KindRemoved, struct {
		ID projects.ID `json:"id"`
	}{ID: id})
	return nil
}

// announceProject publishes a successful change and passes the outcome
// through untouched. A failed change altered nothing, so announcing it would
// be a lie.
func (s *Store) announceProject(
	ctx context.Context, kind string, p projects.Project, err error,
) (projects.Project, error) {
	if err != nil {
		return p, err
	}
	s.publish(ctx, kind, p)
	return p, nil
}

// publish sends one event.
//
// A failure is logged and swallowed: the change already happened, and failing
// the call would report a completed change as an error. A client that misses
// the notification finds out on its next fetch, which is the design's one
// recovery path (ADR-0008).
//
// The whole record travels, unlike settings where only the key does. A project
// record is a name and a path the user chose to share, a list is small, and a
// client that must fetch after every event is polling with extra steps.
func (s *Store) publish(ctx context.Context, kind string, payload any) {
	if s.pub == nil {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("cannot encode a project announcement", "kind", kind, "err", err)
		return
	}
	if err := s.pub.Publish(ctx, bus.Message{Rooms: []bus.Room{Room}, Kind: kind, Data: body}); err != nil {
		slog.Warn("cannot announce a project change", "kind", kind, "err", err)
	}
}
