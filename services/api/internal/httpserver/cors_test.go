package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const webApp = "http://localhost:39180"

func do(t *testing.T, cfg Config, method, path, origin string, preflightFor string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if preflightFor != "" {
		req.Header.Set("Access-Control-Request-Method", preflightFor)
	}
	rec := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rec, req)
	return rec
}

// The bug this exists for: the web app is on another port, so every fetch is
// cross-origin. Without a preflight answer the browser reports only "Load
// failed", with nothing in it to say the request was never made.
func TestPreflightIsAnswered(t *testing.T) {
	cfg := Config{Origins: []string{webApp}}
	rec := do(t, cfg, http.MethodOptions, "/health-check", webApp, http.MethodPut)

	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != webApp {
		t.Errorf("Allow-Origin = %q, want %q", got, webApp)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("preflight named no allowed methods")
	}
}

func TestAllowedOriginCanReadResponses(t *testing.T) {
	rec := do(t, Config{Origins: []string{webApp}}, http.MethodGet, "/health-check", webApp, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != webApp {
		t.Errorf("Allow-Origin = %q, want %q", got, webApp)
	}
}

// An origin nobody allowed still gets the response — CORS is enforced by the
// browser, not the server — but without the header that lets it be read.
func TestUnknownOriginGetsNoHeader(t *testing.T) {
	rec := do(t, Config{Origins: []string{webApp}}, http.MethodGet, "/health-check", "http://evil.test", "")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want none for an unlisted origin", got)
	}
}

func TestNoOriginsConfiguredAllowsNone(t *testing.T) {
	rec := do(t, Config{}, http.MethodGet, "/health-check", webApp, "")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want none when nothing is configured", got)
	}
}

// Caches must not serve one origin's answer to another.
func TestVaryOnOrigin(t *testing.T) {
	rec := do(t, Config{Origins: []string{webApp}}, http.MethodGet, "/health-check", webApp, "")
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin", got)
	}
}

func TestWildcardAllowsAnyOrigin(t *testing.T) {
	rec := do(t, Config{Origins: []string{"*"}}, http.MethodGet, "/health-check", "http://anywhere.test", "")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://anywhere.test" {
		t.Errorf("Allow-Origin = %q, want the request origin echoed", got)
	}
}

// A plain OPTIONS that is not a preflight must not be swallowed.
func TestNonPreflightOptionsFallsThrough(t *testing.T) {
	rec := do(t, Config{Origins: []string{webApp}}, http.MethodOptions, "/health-check", webApp, "")
	if rec.Code == http.StatusNoContent {
		t.Error("a non-preflight OPTIONS was answered as a preflight")
	}
}

// Credentials would make a wildcard origin genuinely dangerous, and nothing
// here uses cookies.
func TestCredentialsAreNeverAllowed(t *testing.T) {
	rec := do(t, Config{Origins: []string{"*"}}, http.MethodGet, "/health-check", webApp, "")
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Allow-Credentials = %q, want it never set", got)
	}
}
