package feature_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpserver"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projectsbus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projectshttp"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// The whole of ADR-0010 assembled the way cmd/server assembles it: one client
// changes a project over the socket, another sees it without asking, and the
// change is equally visible over HTTP.
func TestProjectsStayInSyncAcrossClients(t *testing.T) {
	t.Parallel()

	base, workspace := projectServer(t)
	watcher := listen(t, base, string(projectsbus.Room))
	writer := listen(t, base, "")

	// Created over the socket.
	writeFrame(t, writer, realtimews.Inbound{
		Type: realtimews.KindProjectCreate, ID: "c1", Name: "ACME Website",
	})
	ack := readFrame(t, writer)
	if ack.Type != realtimews.KindAck {
		t.Fatalf("create reply = %+v, want an ack", ack)
	}
	var created projects.Project
	if err := json.Unmarshal(ack.Result, &created); err != nil {
		t.Fatalf("decode %s: %v", ack.Result, err)
	}
	if created.ID == "" || created.Root == "" {
		t.Fatalf("created = %+v, want an id and a root", created)
	}

	// The watcher hears it, carrying the whole record.
	event := readFrame(t, watcher)
	if event.Kind != projectsbus.KindCreated {
		t.Fatalf("watcher got %+v, want project.created", event)
	}
	var announced projects.Project
	if err := json.Unmarshal(event.Data, &announced); err != nil {
		t.Fatalf("decode %s: %v", event.Data, err)
	}
	if announced.ID != created.ID || announced.Name != "ACME Website" {
		t.Errorf("announced = %+v, want the whole record", announced)
	}

	// And HTTP agrees, because both edges call the same store.
	code, body := request(t, http.MethodGet, base+"/projects/", "")
	if code != http.StatusOK {
		t.Fatalf("GET /projects/ = %d", code)
	}
	var list struct {
		Projects []projects.Project `json:"projects"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	if len(list.Projects) != 1 || list.Projects[0].ID != created.ID {
		t.Errorf("list = %+v, want the project just created", list.Projects)
	}

	// Renamed over HTTP; the socket watcher hears that too.
	code, _ = request(t, http.MethodPatch, base+"/projects/"+string(created.ID), `{"name":"Renamed"}`)
	if code != http.StatusOK {
		t.Fatalf("PATCH = %d", code)
	}
	if event := readFrame(t, watcher); event.Kind != projectsbus.KindChanged {
		t.Errorf("watcher got %+v, want project.changed", event)
	}

	// Removed over the socket. The root goes with it.
	writeFrame(t, writer, realtimews.Inbound{
		Type: realtimews.KindProjectRemove, ID: "r1", Project: string(created.ID),
	})
	if got := readFrame(t, writer); got.Type != realtimews.KindAck {
		t.Fatalf("remove reply = %+v, want an ack", got)
	}
	if event := readFrame(t, watcher); event.Kind != projectsbus.KindRemoved {
		t.Errorf("watcher got %+v, want project.removed", event)
	}
	if _, err := os.Stat(created.Root); !os.IsNotExist(err) {
		t.Errorf("the project root survived removal: %v", err)
	}
	if _, err := os.Stat(workspace); err != nil {
		t.Errorf("removal took the workspace with it: %v", err)
	}
}

// projectServer assembles the pieces cmd/server does.
func projectServer(t *testing.T) (base, workspace string) {
	t.Helper()
	workspace = t.TempDir()
	shared, err := settings.NewFS(filepath.Join(workspace, "shared"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	messages := bus.NewMemory()
	registry := projectsbus.New(projects.NewRegistry(projects.Deps{
		Records: shared, Workspace: workspace,
	}), messages)

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
			{Prefix: "/projects/", Handler: projectshttp.New(registry)},
			{Prefix: "/ws", Handler: realtimews.New(hub, realtimews.Options{
				Origins: []string{"*"}, Projects: registry,
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
	return "http://" + srv.Addr, workspace
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
