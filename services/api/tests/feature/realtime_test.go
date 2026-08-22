package feature_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpserver"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingsbus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingshttp"
)

// The whole point of ADR-0008, assembled the way cmd/server assembles it: a
// write over HTTP reaches a listening client with nobody polling for it.
func TestAWriteReachesAListeningClient(t *testing.T) {
	t.Parallel()

	base, messages := realtimeServer(t)

	first := listen(t, base, "settings")
	second := listen(t, base, "settings")

	if code, _ := request(t, http.MethodPut, base+"/settings/agent/model", `{"m":"opus"}`); code != http.StatusNoContent {
		t.Fatalf("PUT = %d, want 204", code)
	}

	// Both clients hear about it, and neither asked.
	for i, sock := range []*websocket.Conn{first, second} {
		frame := readFrame(t, sock)
		if frame.Type != realtimews.KindEvent || frame.Kind != "setting.changed" {
			t.Fatalf("client %d got %+v, want a setting.changed event", i, frame)
		}
		var payload struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(frame.Data, &payload); err != nil {
			t.Fatalf("client %d: decoding %s: %v", i, frame.Data, err)
		}
		if payload.Key != "agent/model" {
			t.Errorf("client %d saw key %q, want agent/model", i, payload.Key)
		}
	}

	// And a delete is announced too.
	if code, _ := request(t, http.MethodDelete, base+"/settings/agent/model", ""); code != http.StatusNoContent {
		t.Fatalf("DELETE = %d, want 204", code)
	}
	if frame := readFrame(t, first); frame.Kind != "setting.deleted" {
		t.Errorf("frame = %+v, want setting.deleted", frame)
	}

	_ = messages
}

// A client that never joined the room hears nothing, which is the "not more"
// half of what rooms are for.
func TestAClientOutsideTheRoomHearsNothing(t *testing.T) {
	t.Parallel()

	base, _ := realtimeServer(t)
	outsider := listen(t, base, "some-other-room")

	if code, _ := request(t, http.MethodPut, base+"/settings/k", `{"v":1}`); code != http.StatusNoContent {
		t.Fatalf("PUT = %d, want 204", code)
	}
	// A pong must arrive before any event would.
	writeFrame(t, outsider, realtimews.Inbound{Type: realtimews.KindPing, ID: "p"})
	if frame := readFrame(t, outsider); frame.Type != realtimews.KindPong {
		t.Errorf("frame = %+v, want only a pong", frame)
	}
}

// realtimeServer assembles the same pieces cmd/server does.
func realtimeServer(t *testing.T) (string, *bus.Memory) {
	t.Helper()

	store, err := settings.NewFS(t.TempDir())
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	messages := bus.NewMemory()
	hub := realtime.New()

	ctx, cancel := context.WithCancel(t.Context())
	pump, err := hub.Attach(ctx, messages)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	done := make(chan struct{})
	go func() { defer close(done); _ = pump() }()

	srv, errCh := httpserver.Start(httpserver.Config{
		Port:     "0",
		Timeouts: config.DefaultServerConfig().HTTP,
		Mounts: []httpserver.Mount{
			{Prefix: "/settings/", Handler: settingshttp.New(settingsbus.New(store, messages))},
			{Prefix: "/ws", Handler: realtimews.New(hub, realtimews.Options{Origins: []string{"*"}})},
		},
	})
	if srv == nil {
		t.Fatalf("Start: %v", <-errCh)
	}
	t.Cleanup(func() {
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = srv.Shutdown(shutdown)
		cancel()
		_ = messages.Close()
		<-done
	})
	return "http://" + srv.Addr, messages
}

// listen dials the socket and joins one room, returning once the join is
// acknowledged so no publish can race it.
func listen(t *testing.T, base, room string) *websocket.Conn {
	t.Helper()
	sock, resp, err := websocket.Dial(t.Context(), "ws"+base[len("http"):]+"/ws", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	closeBody(resp)
	t.Cleanup(func() { _ = sock.CloseNow() })

	if frame := readFrame(t, sock); frame.Type != realtimews.KindWelcome {
		t.Fatalf("first frame = %+v, want welcome", frame)
	}
	writeFrame(t, sock, realtimews.Inbound{Type: realtimews.KindJoin, ID: "join", Room: room})
	if frame := readFrame(t, sock); frame.Type != realtimews.KindAck {
		t.Fatalf("join reply = %+v, want an ack", frame)
	}
	return sock
}

// closeBody releases the handshake response websocket.Dial returns.
func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func writeFrame(t *testing.T, sock *websocket.Conn, in realtimews.Inbound) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, sock, in); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readFrame(t *testing.T, sock *websocket.Conn) realtimews.Outbound {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	var out realtimews.Outbound
	if err := wsjson.Read(ctx, sock, &out); err != nil {
		t.Fatalf("read: %v", err)
	}
	return out
}
