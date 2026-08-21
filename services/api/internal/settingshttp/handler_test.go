package settingshttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingshttp"
)

func serve(t *testing.T) (http.Handler, settings.Store) {
	t.Helper()
	s := settings.NewMemory()
	return settingshttp.New(s, nil), s
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

func TestPutThenGet(t *testing.T) {
	h, _ := serve(t)
	if code := do(t, h, http.MethodPut, "/agent/model", `{"m":"opus"}`).Code; code != http.StatusNoContent {
		t.Fatalf("PUT = %d, want 204", code)
	}
	rec := do(t, h, http.MethodGet, "/agent/model", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET = %d, want 200", rec.Code)
	}
	if rec.Body.String() != `{"m":"opus"}` {
		t.Errorf("GET body = %s, want the stored document", rec.Body.String())
	}
}

func TestGetMissingIs404(t *testing.T) {
	h, _ := serve(t)
	if code := do(t, h, http.MethodGet, "/absent", "").Code; code != http.StatusNotFound {
		t.Errorf("GET absent = %d, want 404", code)
	}
}

func TestBadKeyAndBadDocumentAre400(t *testing.T) {
	h, _ := serve(t)
	for _, path := range []string{"/.hidden", "/bad*char", "/a%20b"} {
		if code := do(t, h, http.MethodGet, path, "").Code; code != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400", path, code)
		}
	}
	if code := do(t, h, http.MethodPut, "/k", "not json").Code; code != http.StatusBadRequest {
		t.Errorf("PUT non-JSON = %d, want 400", code)
	}
}

// Traversal is stopped twice over, and the two layers catch different
// things. ServeMux normalizes a literal "/../" away before routing, so it
// never reaches the store — but a percent-encoded one survives that and is
// refused by the key validator instead. Neither alone is sufficient.
func TestTraversalIsRefused(t *testing.T) {
	h, _ := serve(t)

	for _, path := range []string{"/../escape", "/a/../../etc/passwd"} {
		if code := do(t, h, http.MethodGet, path, "").Code; code != http.StatusTemporaryRedirect {
			t.Errorf("GET %s = %d, want the mux to normalize it away (307)", path, code)
		}
	}
	for _, path := range []string{"/%2e%2e/escape", "/%2e%2e%2fescape"} {
		if code := do(t, h, http.MethodGet, path, "").Code; code != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400: encoded traversal survives path cleaning", path, code)
		}
	}
}

// The claim that matters is not the status code but that nothing is ever
// written outside the root, so this one checks the filesystem itself.
func TestNothingEscapesTheRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	outside := filepath.Join(filepath.Dir(root), "escaped.json")
	h := settingshttp.New(mustFS(t, root), nil)

	for _, path := range []string{
		"/%2e%2e/escaped", "/%2e%2e%2fescaped", "/a/%2e%2e/%2e%2e/escaped", "/../escaped",
	} {
		do(t, h, http.MethodPut, path, `{"pwned":true}`)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("a write escaped the root: %s exists", outside)
	}
}

// An oversized body and an unreadable one are different failures. Both used
// to be reported as 413.
func TestOversizedBodyIs413(t *testing.T) {
	h, _ := serve(t)
	huge := `{"v":"` + strings.Repeat("x", 2<<20) + `"}`
	if code := do(t, h, http.MethodPut, "/k", huge).Code; code != http.StatusRequestEntityTooLarge {
		t.Errorf("PUT of a 2MiB body = %d, want 413", code)
	}
	// A body under the cap is unaffected.
	ok := `{"v":"` + strings.Repeat("x", 1024) + `"}`
	if code := do(t, h, http.MethodPut, "/k", ok).Code; code != http.StatusNoContent {
		t.Errorf("PUT of a small body = %d, want 204", code)
	}
}

func TestDelete(t *testing.T) {
	h, _ := serve(t)
	do(t, h, http.MethodPut, "/k", `{}`)
	if code := do(t, h, http.MethodDelete, "/k", "").Code; code != http.StatusNoContent {
		t.Errorf("DELETE = %d, want 204", code)
	}
	if code := do(t, h, http.MethodDelete, "/k", "").Code; code != http.StatusNotFound {
		t.Errorf("second DELETE = %d, want 404", code)
	}
}

func TestListIsAlwaysAnArray(t *testing.T) {
	h, _ := serve(t)
	var empty struct {
		Keys []string `json:"keys"`
	}
	rec := do(t, h, http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if empty.Keys == nil {
		t.Error("empty listing encoded as null; clients should always get an array")
	}

	do(t, h, http.MethodPut, "/b", `{}`)
	do(t, h, http.MethodPut, "/a", `{}`)
	rec = do(t, h, http.MethodGet, "/", "")
	if !strings.Contains(rec.Body.String(), `["a","b"]`) {
		t.Errorf("listing = %s, want sorted keys", rec.Body.String())
	}
}

// The handler names no storage technology, so the same tests must pass
// against a store backed by real files.
func TestServesAFilesystemStoreIdentically(t *testing.T) {
	h := settingshttp.New(mustFS(t, t.TempDir()), nil)
	if code := do(t, h, http.MethodPut, "/agent/model", `{"m":"opus"}`).Code; code != http.StatusNoContent {
		t.Fatalf("PUT = %d, want 204", code)
	}
	if body := do(t, h, http.MethodGet, "/agent/model", "").Body.String(); body != `{"m":"opus"}` {
		t.Errorf("GET body = %s, want the stored document", body)
	}
}

func mustFS(t *testing.T, dir string) *settings.FS {
	t.Helper()
	s, err := settings.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS(%q): %v", dir, err)
	}
	return s
}
