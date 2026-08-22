package feature_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projectsbus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/rooms"
)

// The whole of ADR-0010 assembled the way cmd/server assembles it: one client
// changes a project over the socket, another sees it without asking, and the
// change is equally visible over HTTP.
func TestProjectsStayInSyncAcrossClients(t *testing.T) {
	t.Parallel()

	base, _ := server(t)
	watcher := listen(t, base, string(rooms.Projects))
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
}
