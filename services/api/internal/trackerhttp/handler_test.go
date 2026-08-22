package trackerhttp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/store"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/trackertest"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/trackerhttp"
)

// theProject is the one project these tests address. Anything else is a
// project that does not exist, which is a case worth having.
const theProject = "test-0001"

type scopes struct{ store tracker.Store }

func (s scopes) Tracker(_ context.Context, id projects.ID) (tracker.Store, error) {
	if id != theProject {
		return nil, projects.ErrNotFound
	}
	return s.store, nil
}

func serve(t *testing.T) (http.Handler, tracker.Store) {
	t.Helper()
	s, err := store.New(t.Context(), store.Deps{})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if err := s.PutItemType(t.Context(), trackertest.BugType()); err != nil {
		t.Fatalf("PutItemType: %v", err)
	}
	mux := http.NewServeMux()
	trackerhttp.Register(mux, scopes{store: s})
	return mux, s
}

func url(suffix string) string { return "/projects/" + theProject + "/tracker/" + suffix }

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func decodeItem(t *testing.T, rec *httptest.ResponseRecorder) tracker.Item {
	t.Helper()
	var item tracker.Item
	if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return item
}

// newBug is the smallest valid create for the fixture type.
func newBug(title string) string {
	return `{"type":"` + string(trackertest.TypeBug) + `","title":"` + title + `",` +
		`"fields":{"` + string(trackertest.FieldSummary) + `":{"kind":"text","value":"x"}}}`
}

func TestCreateReadUpdateDelete(t *testing.T) {
	h, _ := serve(t)

	rec := do(t, h, http.MethodPost, url("items"), newBug("It breaks"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST = %d, want 201: %s", rec.Code, rec.Body)
	}
	created := decodeItem(t, rec)
	if created.Status != trackertest.StatusOpen {
		t.Errorf("Status = %q, want the type's initial status", created.Status)
	}

	if code := do(t, h, http.MethodGet, url("items/"+string(created.ID)), "").Code; code != http.StatusOK {
		t.Errorf("GET one = %d, want 200", code)
	}

	rec = do(t, h, http.MethodPatch,
		url("items/"+string(created.ID))+"?version=1", `{"title":"renamed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200: %s", rec.Code, rec.Body)
	}
	if updated := decodeItem(t, rec); updated.Title != "renamed" || updated.Version != 2 {
		t.Errorf("PATCH = %+v, want the new title at version 2", updated)
	}

	if code := do(t, h, http.MethodDelete, url("items/"+string(created.ID))+"?version=2", "").Code; code != http.StatusNoContent {
		t.Errorf("DELETE = %d, want 204", code)
	}
	if code := do(t, h, http.MethodGet, url("items/"+string(created.ID)), "").Code; code != http.StatusNotFound {
		t.Errorf("GET after DELETE = %d, want 404", code)
	}
}

// A write that does not say what it expects to replace is the overwrite
// compare-and-swap exists to prevent, so it is refused rather than defaulted.
func TestAWriteMustStateItsVersion(t *testing.T) {
	h, _ := serve(t)
	created := decodeItem(t, do(t, h, http.MethodPost, url("items"), newBug("x")))
	path := url("items/" + string(created.ID))

	for name, u := range map[string]string{
		"absent":       path,
		"not a number": path + "?version=soon",
		"negative":     path + "?version=-1",
	} {
		t.Run(name, func(t *testing.T) {
			if code := do(t, h, http.MethodPatch, u, `{"title":"x"}`).Code; code != http.StatusBadRequest {
				t.Errorf("PATCH %s = %d, want 400", u, code)
			}
		})
	}
	rec := do(t, h, http.MethodPatch, path+"?version=99", `{"title":"x"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("PATCH at a stale version = %d, want 409", rec.Code)
	}
}

// The tracker's own refusals must reach a client as the state conflicts they
// are, not as bad requests or server errors.
func TestStateConflictsAre409(t *testing.T) {
	h, _ := serve(t)
	parent := decodeItem(t, do(t, h, http.MethodPost, url("items"), newBug("parent")))
	child := `{"type":"` + string(trackertest.TypeBug) + `","title":"child",` +
		`"parent":"` + string(parent.ID) + `",` +
		`"fields":{"` + string(trackertest.FieldSummary) + `":{"kind":"text","value":"x"}}}`
	if code := do(t, h, http.MethodPost, url("items"), child).Code; code != http.StatusCreated {
		t.Fatalf("creating a child = %d, want 201", code)
	}
	// Deleting a parent that still has children is refused.
	rec := do(t, h, http.MethodDelete, url("items/"+string(parent.ID))+"?version=1", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("DELETE of a parent = %d, want 409: %s", rec.Code, rec.Body)
	}
}

// A move the workflow does not declare is the caller asking for something
// impossible, which is a bad request rather than a conflict.
func TestAnUndeclaredTransitionIs400(t *testing.T) {
	h, _ := serve(t)
	created := decodeItem(t, do(t, h, http.MethodPost, url("items"), newBug("x")))
	rec := do(t, h, http.MethodPatch, url("items/"+string(created.ID))+"?version=1",
		`{"status":"`+string(trackertest.StatusFixed)+`"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("open -> fixed = %d, want 400: %s", rec.Code, rec.Body)
	}
}

func TestAnUnknownProjectIs404(t *testing.T) {
	h, _ := serve(t)
	if code := do(t, h, http.MethodGet, "/projects/ghost-0000/tracker/items", "").Code; code != http.StatusNotFound {
		t.Errorf("GET under an unknown project = %d, want 404", code)
	}
}
