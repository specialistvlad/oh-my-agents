package feature_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/rooms"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/scopes"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// The whole slice assembled as cmd/server assembles it: an item created over
// the socket reaches another client watching the project, is readable over
// HTTP, and is on disk inside the project's own directory.
func TestATrackerItemReachesEveryoneAndLands(t *testing.T) {
	t.Parallel()
	base, _ := server(t)
	p := createProject(t, base, "Tracked")

	watcher := listen(t, base, string(rooms.Project(p.ID)))
	writer := listen(t, base, "")

	body, err := json.Marshal(tracker.NewItem{Type: scopes.StarterType, Title: "Ship it"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	writeFrame(t, writer, realtimews.Inbound{
		Type: realtimews.KindItemCreate, ID: "c1",
		Project: string(p.ID), Body: body,
	})
	ack := readFrame(t, writer)
	if ack.Type != realtimews.KindAck {
		t.Fatalf("create reply = %+v, want an ack", ack)
	}
	var created tracker.Item
	if err := json.Unmarshal(ack.Result, &created); err != nil {
		t.Fatalf("decode %s: %v", ack.Result, err)
	}
	if created.ID == "" || created.Status != scopes.StatusTodo {
		t.Fatalf("created = %+v, want an id and the initial status", created)
	}

	// The other client hears about it without asking.
	event := readFrame(t, watcher)
	if event.Type != realtimews.KindEvent || event.Kind != string(tracker.EventItemCreated) {
		t.Fatalf("watcher got %+v, want item_created", event)
	}

	// HTTP agrees, because both edges call the same store.
	code, listed := request(t, http.MethodGet, base+"/projects/"+string(p.ID)+"/tracker/items", "")
	if code != http.StatusOK {
		t.Fatalf("GET items = %d: %s", code, listed)
	}
	var page struct {
		Items []tracker.Item `json:"items"`
	}
	if err := json.Unmarshal([]byte(listed), &page); err != nil {
		t.Fatalf("decode %s: %v", listed, err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != created.ID {
		t.Errorf("items = %+v, want the one just created", page.Items)
	}

	// And it is on disk, inside this project.
	stored := filepath.Join(p.Root, "tracker", "items", string(created.ID)+".json")
	if _, err := os.Stat(stored); err != nil {
		t.Errorf("the item is not stored at %s: %v", stored, err)
	}
}

// A move the workflow declares works over HTTP and is announced; one it does
// not is refused with the same status the socket would give.
func TestTransitionsAreEnforcedAndAnnounced(t *testing.T) {
	t.Parallel()
	base, _ := server(t)
	p := createProject(t, base, "Workflow")
	items := base + "/projects/" + string(p.ID) + "/tracker/items"

	code, created := request(t, http.MethodPost, items,
		`{"type":"`+string(scopes.StarterType)+`","title":"Move me"}`)
	if code != http.StatusCreated {
		t.Fatalf("POST = %d: %s", code, created)
	}
	var item tracker.Item
	if err := json.Unmarshal([]byte(created), &item); err != nil {
		t.Fatalf("decode: %v", err)
	}
	watcher := listen(t, base, string(rooms.Project(p.ID)))

	one := items + "/" + string(item.ID) + "?version=1"
	code, body := request(t, http.MethodPatch, one, `{"status":"`+string(scopes.StatusDoing)+`"}`)
	if code != http.StatusOK {
		t.Fatalf("declared transition = %d: %s", code, body)
	}
	if got := readFrame(t, watcher); got.Kind != string(tracker.EventStatusChanged) {
		t.Errorf("watcher got %+v, want status_changed", got)
	}

	// todo -> done is not in the starter workflow; doing is the way through.
	two := items + "/" + string(item.ID) + "?version=2"
	code, _ = request(t, http.MethodPatch, two, `{"status":"`+string(scopes.StatusTodo)+`"}`)
	if code != http.StatusOK {
		t.Fatalf("doing -> todo = %d, want 200", code)
	}
	three := items + "/" + string(item.ID) + "?version=3"
	code, body = request(t, http.MethodPatch, three, `{"status":"`+string(scopes.StatusDone)+`"}`)
	if code != http.StatusBadRequest {
		t.Errorf("todo -> done = %d, want 400: %s", code, body)
	}
}

// Trackers are scoped like everything else: one project's items are not
// another's, and each announces to its own room.
func TestTrackersAreScopedToTheirProject(t *testing.T) {
	t.Parallel()
	base, _ := server(t)
	first := createProject(t, base, "First")
	second := createProject(t, base, "Second")

	watchingSecond := listen(t, base, string(rooms.Project(second.ID)))

	code, _ := request(t, http.MethodPost, base+"/projects/"+string(first.ID)+"/tracker/items",
		`{"type":"`+string(scopes.StarterType)+`","title":"only in the first"}`)
	if code != http.StatusCreated {
		t.Fatalf("POST = %d", code)
	}

	_, listed := request(t, http.MethodGet, base+"/projects/"+string(second.ID)+"/tracker/items", "")
	var page struct {
		Items []tracker.Item `json:"items"`
	}
	if err := json.Unmarshal([]byte(listed), &page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("the second project has %d items, want none", len(page.Items))
	}
	// And it heard nothing: a pong must arrive before any event would.
	writeFrame(t, watchingSecond, realtimews.Inbound{Type: realtimews.KindPing, ID: "p"})
	if got := readFrame(t, watchingSecond); got.Type != realtimews.KindPong {
		t.Errorf("the second project's watcher got %+v, want only a pong", got)
	}
}
