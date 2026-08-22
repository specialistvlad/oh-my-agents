package realtimews_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/coder/websocket"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

func projectSocket(t *testing.T) *websocket.Conn {
	t.Helper()
	workspace := t.TempDir()
	records, err := settings.NewFS(filepath.Join(workspace, "shared"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	store := projects.NewRegistry(projects.Deps{Records: records, Workspace: workspace})

	srv := httptest.NewServer(realtimews.New(realtime.New(), realtimews.Options{
		Origins: []string{"*"}, Projects: store,
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
	return sock
}

func TestCreateOverTheSocketReturnsTheProject(t *testing.T) {
	sock := projectSocket(t)
	send(t, sock, realtimews.Inbound{
		Type: realtimews.KindProjectCreate, ID: "c1", Name: "ACME Website",
	})
	got := recv(t, sock)
	if got.Type != realtimews.KindAck {
		t.Fatalf("reply = %+v, want an ack", got)
	}
	var p projects.Project
	if err := json.Unmarshal(got.Result, &p); err != nil {
		t.Fatalf("decode %s: %v", got.Result, err)
	}
	if p.ID == "" || p.Root == "" || p.Name != "ACME Website" {
		t.Errorf("result = %+v, want the created project", p)
	}
}

// A client that never saw its acknowledgement replays after reconnecting. It
// must get back what the first attempt produced, not an empty acknowledgement
// — otherwise the retrying client ends up worse off than one that never lost
// its connection, with no idea what it created.
func TestReplayedCreateReturnsTheOriginalProject(t *testing.T) {
	sock := projectSocket(t)
	command := realtimews.Inbound{
		Type: realtimews.KindProjectCreate, ID: "c1",
		Name: "Once Only", Idempotency: "the-same-key",
	}
	send(t, sock, command)
	first := recv(t, sock)
	if first.Type != realtimews.KindAck || len(first.Result) == 0 {
		t.Fatalf("first reply = %+v, want an ack carrying the project", first)
	}

	command.ID = "c2" // a different correlation id; the same command
	send(t, sock, command)
	replay := recv(t, sock)
	if replay.Type != realtimews.KindAck || replay.ID != "c2" {
		t.Fatalf("replay = %+v, want an ack echoing c2", replay)
	}
	if string(replay.Result) != string(first.Result) {
		t.Errorf("replay result = %s, want the original %s", replay.Result, first.Result)
	}
}

func TestProjectFailuresCarryHTTPStatuses(t *testing.T) {
	sock := projectSocket(t)
	for name, c := range map[string]struct {
		in   realtimews.Inbound
		want int
	}{
		"empty name": {realtimews.Inbound{Type: realtimews.KindProjectCreate, ID: "1"}, http.StatusBadRequest},
		"bad id": {
			realtimews.Inbound{Type: realtimews.KindProjectRename, ID: "2", Project: "NOT VALID", Name: "x"},
			http.StatusBadRequest,
		},
		"absent": {
			realtimews.Inbound{Type: realtimews.KindProjectRemove, ID: "3", Project: "absent-0000"},
			http.StatusNotFound,
		},
	} {
		t.Run(name, func(t *testing.T) {
			send(t, sock, c.in)
			got := recv(t, sock)
			if got.Type != realtimews.KindError || got.Status != c.want {
				t.Errorf("reply = %+v, want an error with status %d", got, c.want)
			}
		})
	}
}

func TestASocketWithoutProjectsRefusesThem(t *testing.T) {
	srv := httptest.NewServer(realtimews.New(realtime.New(), realtimews.Options{Origins: []string{"*"}}))
	t.Cleanup(srv.Close)
	sock, resp, err := websocket.Dial(t.Context(), "ws"+srv.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	closeBody(resp)
	t.Cleanup(func() { _ = sock.CloseNow() })
	recv(t, sock)

	send(t, sock, realtimews.Inbound{Type: realtimews.KindProjectCreate, ID: "1", Name: "x"})
	if got := recv(t, sock); got.Type != realtimews.KindError {
		t.Errorf("reply = %+v, want an error", got)
	}
}
