package settingshttp

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// SettingsRoom is where settings activity is published. It is a single room
// today; ADR-0009 scopes it per project once projects exist.
const SettingsRoom bus.Room = "settings"

// BusAnnouncer publishes settings writes onto a bus.
type BusAnnouncer struct {
	Bus bus.Publisher
}

// Announce implements [Announcer]. A failure to publish is logged and
// swallowed: the write already succeeded, and failing the request would
// report a durable change as an error. A client that misses the notification
// finds out on its next read or reconnect, which is the design's one recovery
// path (ADR-0008).
func (a BusAnnouncer) Announce(ctx context.Context, kind string, key settings.Key) {
	payload, err := json.Marshal(struct {
		Key settings.Key `json:"key"`
	}{Key: key})
	if err != nil {
		slog.Warn("cannot encode a settings announcement", "key", key, "err", err)
		return
	}
	message := bus.Message{Rooms: []bus.Room{SettingsRoom}, Kind: kind, Data: payload}
	if err := a.Bus.Publish(ctx, message); err != nil {
		slog.Warn("cannot announce a settings write", "key", key, "err", err)
	}
}
