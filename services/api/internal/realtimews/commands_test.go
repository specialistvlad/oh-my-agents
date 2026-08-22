package realtimews_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/rooms"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingsbus"
)

// theProject is the project these mutations address.
const theProject = "test-0001"

// scoped hands out one store for one project, refusing every other id the way
// a real registry does.
type scoped struct{ store settings.Store }

func (s scoped) Settings(_ context.Context, id projects.ID) (settings.Store, error) {
	if id != theProject {
		return nil, projects.ErrNotFound
	}
	return s.store, nil
}

// writable starts a server whose socket accepts mutations, and returns a
// dialed client plus the store behind it.
func writable(t *testing.T) (*websocket.Conn, settings.Store) {
	t.Helper()
	b := bus.NewMemory()
	t.Cleanup(func() { _ = b.Close() })
	store := settingsbus.New(settings.NewMemory(), b, "project:test")

	h := realtime.New()
	srv := httptest.NewServer(realtimews.New(h, realtimews.Options{
		Origins:  []string{"*"},
		Settings: scoped{store: store},
	}))
	t.Cleanup(srv.Close)

	sock, resp, err := websocket.Dial(t.Context(), "ws"+srv.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	closeBody(resp)
	t.Cleanup(func() { _ = sock.CloseNow() })
	if got := recv(t, sock); got.Type != realtimews.KindWelcome {
		t.Fatalf("first frame = %+v, want welcome", got)
	}
	return sock, store
}

func TestWriteOverTheSocket(t *testing.T) {
	sock, store := writable(t)

	send(t, sock, realtimews.Inbound{
		Project: theProject,
		Type:    realtimews.KindSet, ID: "w1",
		Key: "agent/model", Value: json.RawMessage(`{"m":"opus"}`),
	})
	if got := recv(t, sock); got.Type != realtimews.KindAck || got.ID != "w1" {
		t.Fatalf("reply = %+v, want an ack echoing w1", got)
	}
	doc, err := store.Get(t.Context(), "agent/model")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(doc) != `{"m":"opus"}` {
		t.Errorf("stored %s, want the document as sent", doc)
	}
}

func TestDeleteOverTheSocket(t *testing.T) {
	sock, store := writable(t)
	if err := store.Set(t.Context(), "k", settings.Document(`{}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	send(t, sock, realtimews.Inbound{Project: theProject, Type: realtimews.KindDelete, ID: "d1", Key: "k"})
	if got := recv(t, sock); got.Type != realtimews.KindAck {
		t.Fatalf("reply = %+v, want an ack", got)
	}
	if _, err := store.Get(t.Context(), "k"); err == nil {
		t.Error("the setting is still there after a delete over the socket")
	}
}

// The reason idempotency keys exist: a client that never saw an ack replays
// after reconnecting, and a delete applied twice reports a failure for
// something that in fact succeeded.
func TestReplayingADeleteIsAcknowledgedNotRefused(t *testing.T) {
	sock, store := writable(t)
	if err := store.Set(t.Context(), "k", settings.Document(`{}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	command := realtimews.Inbound{
		Project: theProject,
		Type:    realtimews.KindDelete, ID: "d1", Key: "k", Idempotency: "once-only",
	}
	send(t, sock, command)
	if got := recv(t, sock); got.Type != realtimews.KindAck {
		t.Fatalf("first delete = %+v, want an ack", got)
	}

	command.ID = "d2" // a different correlation id; the same command
	send(t, sock, command)
	if got := recv(t, sock); got.Type != realtimews.KindAck || got.ID != "d2" {
		t.Errorf("replayed delete = %+v, want an ack echoing d2", got)
	}
}

// Without a key, a command is applied every time — which is what a client
// asking for that should get.
func TestWithoutAKeyACommandRunsEveryTime(t *testing.T) {
	sock, store := writable(t)
	if err := store.Set(t.Context(), "k", settings.Document(`{}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	del := realtimews.Inbound{Project: theProject, Type: realtimews.KindDelete, ID: "d1", Key: "k"}
	send(t, sock, del)
	if got := recv(t, sock); got.Type != realtimews.KindAck {
		t.Fatalf("first delete = %+v, want an ack", got)
	}
	send(t, sock, del)
	if got := recv(t, sock); got.Type != realtimews.KindError || got.Status != http.StatusNotFound {
		t.Errorf("second delete = %+v, want a 404 error", got)
	}
}

// Both edges must describe one failure the same way, or a client has to learn
// two vocabularies for the same store.
func TestFailuresCarryTheStatusHTTPWouldHaveGiven(t *testing.T) {
	sock, _ := writable(t)
	for _, c := range []struct {
		name string
		in   realtimews.Inbound
		want int
	}{
		{"bad key", realtimews.Inbound{Project: theProject, Type: realtimews.KindSet, ID: "1", Key: "../escape", Value: json.RawMessage(`{}`)}, http.StatusBadRequest},
		// A malformed document cannot be expressed here: the value is a
		// JSON value inside a JSON frame, so a frame carrying one would
		// not parse at all. Absent is the only invalid document the
		// socket can produce — unlike HTTP, where the body is opaque
		// bytes and `nope` reaches the store intact.
		{"no document", realtimews.Inbound{Project: theProject, Type: realtimews.KindSet, ID: "2", Key: "k"}, http.StatusBadRequest},
		{"absent", realtimews.Inbound{Project: theProject, Type: realtimews.KindDelete, ID: "3", Key: "gone"}, http.StatusNotFound},
	} {
		t.Run(c.name, func(t *testing.T) {
			send(t, sock, c.in)
			got := recv(t, sock)
			if got.Type != realtimews.KindError || got.Status != c.want {
				t.Errorf("reply = %+v, want an error with status %d", got, c.want)
			}
		})
	}
}

func TestAMissingKeyIsRefused(t *testing.T) {
	sock, _ := writable(t)
	send(t, sock, realtimews.Inbound{Project: theProject, Type: realtimews.KindSet, ID: "1", Value: json.RawMessage(`{}`)})
	if got := recv(t, sock); got.Type != realtimews.KindError {
		t.Errorf("reply = %+v, want an error", got)
	}
}

// A socket with no write surface says so rather than pretending.
func TestAReadOnlySocketRefusesWrites(t *testing.T) {
	h := realtime.New()
	srv := httptest.NewServer(realtimews.New(h, realtimews.Options{Origins: []string{"*"}}))
	t.Cleanup(srv.Close)

	sock, resp, err := websocket.Dial(t.Context(), "ws"+srv.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	closeBody(resp)
	t.Cleanup(func() { _ = sock.CloseNow() })
	recv(t, sock) // welcome

	send(t, sock, realtimews.Inbound{Project: theProject, Type: realtimews.KindSet, ID: "1", Key: "k", Value: json.RawMessage(`{}`)})
	if got := recv(t, sock); got.Type != realtimews.KindError {
		t.Errorf("reply = %+v, want an error", got)
	}
}

// A write over the socket must reach other clients exactly as an HTTP one
// does — that is the whole reason announcing moved into the store.
func TestASocketWriteIsAnnouncedToOtherClients(t *testing.T) {
	b := bus.NewMemory()
	t.Cleanup(func() { _ = b.Close() })
	// The room a project's settings announce to (ADR-0009), which is what a
	// watcher has to join to hear them.
	room := rooms.Project(theProject)
	store := settingsbus.New(settings.NewMemory(), b, room)

	h := realtime.New()
	ctx := t.Context()
	pump, err := h.Attach(ctx, b)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	go func() { _ = pump() }()

	srv := httptest.NewServer(realtimews.New(h, realtimews.Options{Origins: []string{"*"}, Settings: scoped{store: store}}))
	t.Cleanup(srv.Close)
	url := "ws" + srv.URL[len("http"):]

	dial := func() *websocket.Conn {
		sock, resp, err := websocket.Dial(ctx, url, nil)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		closeBody(resp)
		t.Cleanup(func() { _ = sock.CloseNow() })
		recv(t, sock)
		return sock
	}
	writer, watcher := dial(), dial()

	send(t, watcher, realtimews.Inbound{Type: realtimews.KindJoin, ID: "j", Room: string(room)})
	if got := recv(t, watcher); got.Type != realtimews.KindAck {
		t.Fatalf("join reply = %+v, want an ack", got)
	}
	send(t, writer, realtimews.Inbound{
		Project: theProject,
		Type:    realtimews.KindSet, ID: "w", Key: "shared", Value: json.RawMessage(`{"v":1}`),
	})
	if got := recv(t, writer); got.Type != realtimews.KindAck {
		t.Fatalf("write reply = %+v, want an ack", got)
	}
	if got := recv(t, watcher); got.Type != realtimews.KindEvent || got.Kind != settingsbus.KindChanged {
		t.Errorf("watcher got %+v, want a setting.changed event", got)
	}
}
