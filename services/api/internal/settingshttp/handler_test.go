package settingshttp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingshttp"
)

// theProject is the one project these tests address. Anything else is a
// project that does not exist, which is a case worth having.
const theProject = "test-0001"

// scopes hands out one store for one project, and refuses every other id the
// way a real registry would.
type scopes struct{ store settings.Store }

func (s scopes) Settings(_ context.Context, id projects.ID) (settings.Store, error) {
	if id != theProject {
		return nil, projects.ErrNotFound
	}
	return s.store, nil
}

func serve(t *testing.T) (http.Handler, settings.Store) {
	t.Helper()
	return serveStore(t, settings.NewMemory())
}

func serveStore(t *testing.T, store settings.Store) (http.Handler, settings.Store) {
	t.Helper()
	mux := http.NewServeMux()
	settingshttp.Register(mux, scopes{store: store})
	return mux, store
}

// path builds a settings URL for the project under test.
func path(key string) string { return "/projects/" + theProject + "/settings/" + key }

func do(t *testing.T, h http.Handler, method, url, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, url, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, url, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestPutThenGet(t *testing.T) {
	h, _ := serve(t)
	if code := do(t, h, http.MethodPut, path("agent/model"), `{"m":"opus"}`).Code; code != http.StatusNoContent {
		t.Fatalf("PUT = %d, want 204", code)
	}
	rec := do(t, h, http.MethodGet, path("agent/model"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET = %d, want 200", rec.Code)
	}
	if rec.Body.String() != `{"m":"opus"}` {
		t.Errorf("GET body = %s, want the stored document", rec.Body.String())
	}
}

// The point of ADR-0009: settings belong to a project, and a project that
// does not exist has none.
func TestAnUnknownProjectIs404(t *testing.T) {
	h, _ := serve(t)
	for _, url := range []string{
		"/projects/other-0002/settings/k",
		"/projects/other-0002/settings/",
	} {
		if code := do(t, h, http.MethodGet, url, "").Code; code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", url, code)
		}
	}
	if code := do(t, h, http.MethodPut, "/projects/other-0002/settings/k", `{}`).Code; code != http.StatusNotFound {
		t.Errorf("PUT to an unknown project = %d, want 404", code)
	}
}

func TestGetMissingIs404(t *testing.T) {
	h, _ := serve(t)
	if code := do(t, h, http.MethodGet, path("absent"), "").Code; code != http.StatusNotFound {
		t.Errorf("GET absent = %d, want 404", code)
	}
}

func TestBadKeyAndBadDocumentAre400(t *testing.T) {
	h, _ := serve(t)
	for _, key := range []string{".hidden", "bad*char", "a%20b"} {
		if code := do(t, h, http.MethodGet, path(key), "").Code; code != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400", key, code)
		}
	}
	if code := do(t, h, http.MethodPut, path("k"), "not json").Code; code != http.StatusBadRequest {
		t.Errorf("PUT non-JSON = %d, want 400", code)
	}
}

// Traversal is stopped twice over, and the layers catch different things.
// ServeMux normalizes a literal "/../" away before routing; a percent-encoded
// one survives that and is refused by the key grammar.
func TestTraversalIsRefused(t *testing.T) {
	h, _ := serve(t)
	for _, key := range []string{"../escape", "a/../../etc/passwd"} {
		if code := do(t, h, http.MethodGet, path(key), "").Code; code != http.StatusTemporaryRedirect {
			t.Errorf("GET %s = %d, want the mux to normalize it away (307)", key, code)
		}
	}
	for _, key := range []string{"%2e%2e/escape", "%2e%2e%2fescape"} {
		if code := do(t, h, http.MethodGet, path(key), "").Code; code != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400: encoded traversal survives path cleaning", key, code)
		}
	}
}

// The claim that matters is not the status code but that nothing is written
// outside the project's own directory.
func TestNothingEscapesTheProjectRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	outside := filepath.Join(filepath.Dir(root), "escaped.json")
	store, err := settings.NewFS(root)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	h, _ := serveStore(t, store)

	for _, key := range []string{"%2e%2e/escaped", "%2e%2e%2fescaped", "a/%2e%2e/%2e%2e/escaped", "../escaped"} {
		do(t, h, http.MethodPut, path(key), `{"pwned":true}`)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("a write escaped the project root: %s exists", outside)
	}
}

func TestDelete(t *testing.T) {
	h, _ := serve(t)
	do(t, h, http.MethodPut, path("k"), `{}`)
	if code := do(t, h, http.MethodDelete, path("k"), "").Code; code != http.StatusNoContent {
		t.Errorf("DELETE = %d, want 204", code)
	}
	if code := do(t, h, http.MethodDelete, path("k"), "").Code; code != http.StatusNotFound {
		t.Errorf("second DELETE = %d, want 404", code)
	}
}

// An oversized body and an unreadable one are different failures.
func TestOversizedBodyIs413(t *testing.T) {
	h, _ := serve(t)
	huge := `{"v":"` + strings.Repeat("x", 2<<20) + `"}`
	if code := do(t, h, http.MethodPut, path("k"), huge).Code; code != http.StatusRequestEntityTooLarge {
		t.Errorf("PUT of a 2MiB body = %d, want 413", code)
	}
	ok := `{"v":"` + strings.Repeat("x", 1024) + `"}`
	if code := do(t, h, http.MethodPut, path("k"), ok).Code; code != http.StatusNoContent {
		t.Errorf("PUT of a small body = %d, want 204", code)
	}
}

func TestListIsAlwaysAnArray(t *testing.T) {
	h, _ := serve(t)
	var empty struct {
		Keys []string `json:"keys"`
	}
	rec := do(t, h, http.MethodGet, path(""), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET list = %d, want 200", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if empty.Keys == nil {
		t.Error("empty listing encoded as null; clients should always get an array")
	}

	do(t, h, http.MethodPut, path("b"), `{}`)
	do(t, h, http.MethodPut, path("a"), `{}`)
	if body := do(t, h, http.MethodGet, path(""), "").Body.String(); !strings.Contains(body, `["a","b"]`) {
		t.Errorf("listing = %s, want sorted keys", body)
	}
}
