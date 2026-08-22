package projectshttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projectshttp"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

func serve(t *testing.T) (http.Handler, projects.Store) {
	t.Helper()
	workspace := t.TempDir()
	records, err := settings.NewFS(filepath.Join(workspace, "shared"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	store := projects.NewRegistry(projects.Deps{Records: records, Workspace: workspace})
	mux := http.NewServeMux()
	projectshttp.Register(mux, store)
	return mux, store
}

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

func decodeProject(t *testing.T, rec *httptest.ResponseRecorder) projects.Project {
	t.Helper()
	var p projects.Project
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return p
}

func TestCreateReadListDelete(t *testing.T) {
	h, _ := serve(t)

	rec := do(t, h, http.MethodPost, "/projects/", `{"name":"ACME Website"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST = %d, want 201: %s", rec.Code, rec.Body)
	}
	created := decodeProject(t, rec)

	if code := do(t, h, http.MethodGet, "/projects/"+string(created.ID), "").Code; code != http.StatusOK {
		t.Errorf("GET one = %d, want 200", code)
	}

	var list struct {
		Projects []projects.Project `json:"projects"`
	}
	rec = do(t, h, http.MethodGet, "/projects/", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Projects) != 1 {
		t.Errorf("list = %d projects, want 1", len(list.Projects))
	}

	if code := do(t, h, http.MethodDelete, "/projects/"+string(created.ID), "").Code; code != http.StatusNoContent {
		t.Errorf("DELETE = %d, want 204", code)
	}
	if code := do(t, h, http.MethodGet, "/projects/"+string(created.ID), "").Code; code != http.StatusNotFound {
		t.Errorf("GET after DELETE = %d, want 404", code)
	}
}

// An empty listing must be an array, so a client never has to handle null.
func TestEmptyListIsAnArray(t *testing.T) {
	h, _ := serve(t)
	rec := do(t, h, http.MethodGet, "/projects/", "")
	if !strings.Contains(rec.Body.String(), `"projects":[]`) {
		t.Errorf("body = %s, want an empty array", rec.Body.String())
	}
}

func TestPatchRenamesAndRepoints(t *testing.T) {
	h, _ := serve(t)
	created := decodeProject(t, do(t, h, http.MethodPost, "/projects/", `{"name":"Before"}`))
	elsewhere := filepath.Join(t.TempDir(), "moved")

	rec := do(t, h, http.MethodPatch, "/projects/"+string(created.ID),
		`{"name":"After","root":"`+elsewhere+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200: %s", rec.Code, rec.Body)
	}
	changed := decodeProject(t, rec)
	if changed.Name != "After" || changed.Root != elsewhere {
		t.Errorf("PATCH = %+v, want the new name and root", changed)
	}
	if changed.ID != created.ID {
		t.Errorf("the id changed: %q -> %q", created.ID, changed.ID)
	}
}

func TestPatchWithNothingToChange(t *testing.T) {
	h, _ := serve(t)
	created := decodeProject(t, do(t, h, http.MethodPost, "/projects/", `{"name":"Idle"}`))
	if code := do(t, h, http.MethodPatch, "/projects/"+string(created.ID), `{}`).Code; code != http.StatusBadRequest {
		t.Errorf("empty PATCH = %d, want 400", code)
	}
}

func TestBadInputIs400(t *testing.T) {
	h, _ := serve(t)
	for name, body := range map[string]string{
		"empty name": `{"name":""}`,
		"root at /":  `{"name":"ok","root":"/"}`,
		"not json":   `nonsense`,
	} {
		t.Run(name, func(t *testing.T) {
			if code := do(t, h, http.MethodPost, "/projects/", body).Code; code != http.StatusBadRequest {
				t.Errorf("POST %s = %d, want 400", body, code)
			}
		})
	}
	// Percent-encoded, because a raw space cannot appear in a request line.
	// The mux decodes it, so the store still sees the malformed id.
	if code := do(t, h, http.MethodGet, "/projects/Not%20A%20Valid%20Id", "").Code; code != http.StatusBadRequest {
		t.Errorf("GET with a malformed id = %d, want 400", code)
	}
}

// The marker refusal is a conflict, not a bad request: the call was fine and
// the directory is not what the record says it is.
func TestRemovingAnUnmarkedRootIs409(t *testing.T) {
	h, store := serve(t)
	created := decodeProject(t, do(t, h, http.MethodPost, "/projects/", `{"name":"Guarded"}`))

	p, err := store.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := os.Remove(filepath.Join(p.Root, projects.MarkerName)); err != nil {
		t.Fatalf("removing the marker: %v", err)
	}
	rec := do(t, h, http.MethodDelete, "/projects/"+string(created.ID), "")
	if rec.Code != http.StatusConflict {
		t.Errorf("DELETE of an unmarked root = %d, want 409", rec.Code)
	}
	if _, err := os.Stat(p.Root); err != nil {
		t.Errorf("a refused delete removed the directory anyway: %v", err)
	}
}
