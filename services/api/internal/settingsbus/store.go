// Package settingsbus announces settings writes on a bus.
//
// It wraps a [settings.Store] and implements the same interface, so callers
// cannot tell the difference and nothing above it changes. The announcement
// therefore happens at the store, not at an edge: with an HTTP API and a
// WebSocket both able to write, an edge that announces its own writes is one
// that stays silent about everybody else's.
//
// Composition rather than a method on the store, because publishing is not
// storage's job (ADR-0001), and settings knows nothing about a bus.
package settingsbus

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// Event kinds published on a write.
const (
	KindChanged = "setting.changed"
	KindDeleted = "setting.deleted"
)

// Store is a [settings.Store] that announces what it writes.
type Store struct {
	inner settings.Store
	pub   bus.Publisher
	room  bus.Room
}

// New wraps a store. A nil publisher is valid and announces nothing, so a
// process with no realtime surface wires the same way.
// The room is a parameter rather than a constant because settings are scoped
// to a project now (ADR-0009), so their announcements belong to that
// project's room rather than to one global one.
func New(inner settings.Store, pub bus.Publisher, room bus.Room) *Store {
	return &Store{inner: inner, pub: pub, room: room}
}

// Get implements [settings.Reader].
func (s *Store) Get(ctx context.Context, key settings.Key) (settings.Document, error) {
	return s.inner.Get(ctx, key)
}

// Keys implements [settings.Lister].
func (s *Store) Keys(ctx context.Context) ([]settings.Key, error) {
	return s.inner.Keys(ctx)
}

// Set implements [settings.Writer], announcing a successful write.
func (s *Store) Set(ctx context.Context, key settings.Key, doc settings.Document) error {
	if err := s.inner.Set(ctx, key, doc); err != nil {
		return err
	}
	s.announce(ctx, KindChanged, key)
	return nil
}

// Delete implements [settings.Writer], announcing a successful removal.
func (s *Store) Delete(ctx context.Context, key settings.Key) error {
	if err := s.inner.Delete(ctx, key); err != nil {
		return err
	}
	s.announce(ctx, KindDeleted, key)
	return nil
}

// announce publishes one write.
//
// A failure to publish is logged and swallowed. The write already succeeded
// and is durable; failing the call would report a completed change as an
// error. A client that misses the notification finds out on its next fetch,
// which is the design's one recovery path (ADR-0008).
//
// Only the key is published, never the value. Settings hold credentials, and
// a payload on a bus reaches every connected client and the log.
func (s *Store) announce(ctx context.Context, kind string, key settings.Key) {
	if s.pub == nil {
		return
	}
	payload, err := json.Marshal(struct {
		Key settings.Key `json:"key"`
	}{Key: key})
	if err != nil {
		slog.Warn("cannot encode a settings announcement", "key", key, "err", err)
		return
	}
	if err := s.pub.Publish(ctx, bus.Message{Rooms: []bus.Room{s.room}, Kind: kind, Data: payload}); err != nil {
		slog.Warn("cannot announce a settings write", "key", key, "err", err)
	}
}
