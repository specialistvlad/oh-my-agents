package scopes

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/rooms"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/fs"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/store"
)

// Tracker returns the tracker for one project, rooted at its own directory
// and announcing to its own room.
//
// Unlike a settings store, this one is **cached per project and shared**. A
// tracker holds its state in memory and writes through, so two instances over
// one directory would each believe they had the whole story: their sequence
// numbers would collide, and a write through one would be invisible to the
// other until a restart. One per project is a correctness requirement, not a
// performance choice.
func (s *Scopes) Tracker(ctx context.Context, id projects.ID) (tracker.Store, error) {
	p, err := s.registry.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if held, open := s.trackers[p.ID]; open {
		return held, nil
	}
	opened, err := fs.New(ctx, filepath.Join(p.Root, "tracker"), store.Deps{
		Announce: s.announcer(p.ID),
	})
	if err != nil {
		return nil, err
	}
	if err := seed(ctx, opened); err != nil {
		return nil, err
	}
	s.trackers[p.ID] = opened
	return opened, nil
}

// announcer publishes a tracker event to the project's room.
//
// A failure is logged and swallowed. The change is already durable, and
// failing the write to report that nobody heard about it would be worse; a
// client that misses one finds out on its next fetch (ADR-0008).
func (s *Scopes) announcer(id projects.ID) func(context.Context, tracker.Event) {
	if s.pub == nil {
		return nil
	}
	room := rooms.Project(id)
	return func(ctx context.Context, e tracker.Event) {
		body, err := json.Marshal(e)
		if err != nil {
			slog.Warn("cannot encode a tracker event", "seq", e.Seq, "err", err)
			return
		}
		message := bus.Message{Rooms: []bus.Room{room}, Kind: string(e.Kind), Data: body}
		if err := s.pub.Publish(ctx, message); err != nil {
			slog.Warn("cannot announce a tracker event", "seq", e.Seq, "err", err)
		}
	}
}
