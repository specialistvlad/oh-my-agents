package feature_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpapi"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpserver"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projectsbus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/rooms"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/scopes"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// server assembles the pieces cmd/server assembles, and returns its base URL.
func server(t *testing.T) (string, *bus.Memory) {
	t.Helper()
	workspace := t.TempDir()
	shared, err := settings.NewFS(workspace + "/shared")
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	messages := bus.NewMemory()
	registry := projectsbus.New(projects.NewRegistry(projects.Deps{
		Records: shared, Workspace: workspace,
	}), messages)
	scoped := scopes.New(registry, messages)

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
			{Prefix: "/", Handler: httpapi.New(httpapi.Deps{Projects: registry, Scopes: scoped})},
			{Prefix: "/ws", Handler: realtimews.New(hub, realtimews.Options{
				Origins: []string{"*"}, Settings: scoped, Projects: registry,
			})},
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

// The shape ADR-0009 asks for: a setting belongs to a project, its
// announcement goes to that project's room, and a client watching one project
// hears nothing about another.
func TestSettingsAreScopedToTheirProject(t *testing.T) {
	t.Parallel()
	base, _ := server(t)

	first := createProject(t, base, "First")
	second := createProject(t, base, "Second")

	watchingFirst := listen(t, base, string(rooms.Project(first.ID)))
	watchingSecond := listen(t, base, string(rooms.Project(second.ID)))

	code, _ := request(t, http.MethodPut, base+"/projects/"+string(first.ID)+"/settings/agent/model", `{"m":"opus"}`)
	if code != http.StatusNoContent {
		t.Fatalf("PUT = %d, want 204", code)
	}

	// The project it belongs to hears about it...
	frame := readFrame(t, watchingFirst)
	if frame.Type != realtimews.KindEvent || frame.Kind != "setting.changed" {
		t.Fatalf("watcher of the first project got %+v, want setting.changed", frame)
	}
	// ...and the other one does not.
	writeFrame(t, watchingSecond, realtimews.Inbound{Type: realtimews.KindPing, ID: "p"})
	if got := readFrame(t, watchingSecond); got.Type != realtimews.KindPong {
		t.Errorf("watcher of the second project got %+v, want only a pong", got)
	}

	// The value is readable under its own project and absent under the other.
	code, body := request(t, http.MethodGet, base+"/projects/"+string(first.ID)+"/settings/agent/model", "")
	if code != http.StatusOK || body != `{"m":"opus"}` {
		t.Errorf("GET under its own project = %d %s", code, body)
	}
	code, _ = request(t, http.MethodGet, base+"/projects/"+string(second.ID)+"/settings/agent/model", "")
	if code != http.StatusNotFound {
		t.Errorf("GET under another project = %d, want 404", code)
	}
}

// A write over the socket is scoped the same way and reaches the same room,
// because both edges resolve the project through the same scopes.
func TestASocketWriteIsScopedToo(t *testing.T) {
	t.Parallel()
	base, _ := server(t)
	p := createProject(t, base, "Socket")

	watcher := listen(t, base, string(rooms.Project(p.ID)))
	writer := listen(t, base, "")

	writeFrame(t, writer, realtimews.Inbound{
		Type: realtimews.KindSet, ID: "w1", Project: string(p.ID),
		Key: "agent/model", Value: json.RawMessage(`{"m":"opus"}`),
	})
	if got := readFrame(t, writer); got.Type != realtimews.KindAck {
		t.Fatalf("write reply = %+v, want an ack", got)
	}
	if got := readFrame(t, watcher); got.Kind != "setting.changed" {
		t.Errorf("watcher got %+v, want setting.changed", got)
	}
}

// Naming a project that does not exist is a 404 on both edges, so a client
// learns the same thing whichever it used.
func TestAnUnknownProjectIsRefusedOnBothEdges(t *testing.T) {
	t.Parallel()
	base, _ := server(t)

	code, _ := request(t, http.MethodGet, base+"/projects/ghost-0000/settings/k", "")
	if code != http.StatusNotFound {
		t.Errorf("HTTP = %d, want 404", code)
	}
	sock := listen(t, base, "")
	writeFrame(t, sock, realtimews.Inbound{
		Type: realtimews.KindSet, ID: "w", Project: "ghost-0000",
		Key: "k", Value: json.RawMessage(`{}`),
	})
	got := readFrame(t, sock)
	if got.Type != realtimews.KindError || got.Status != http.StatusNotFound {
		t.Errorf("socket = %+v, want a 404 error", got)
	}
}

func createProject(t *testing.T, base, name string) projects.Project {
	t.Helper()
	code, body := request(t, http.MethodPost, base+"/projects/", `{"name":"`+name+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("POST /projects/ = %d: %s", code, body)
	}
	var p projects.Project
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return p
}

func request(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out)
}

// listen dials and, when a room is named, joins it before returning so no
// publish can race the subscription.
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
	if room == "" {
		return sock
	}
	writeFrame(t, sock, realtimews.Inbound{Type: realtimews.KindJoin, ID: "join", Room: room})
	if frame := readFrame(t, sock); frame.Type != realtimews.KindAck {
		t.Fatalf("join reply = %+v, want an ack", frame)
	}
	return sock
}

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
