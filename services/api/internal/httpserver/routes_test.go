package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, cfg Config, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	newHandler(cfg).ServeHTTP(rec, req)
	return rec
}

func TestHealthCheckOK(t *testing.T) {
	rec := get(t, Config{}, "/health-check")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("body = %q, want OK", rec.Body.String())
	}
}

func TestPrefixMountsRoutes(t *testing.T) {
	cfg := Config{Prefix: "/api"}
	if code := get(t, cfg, "/api/health-check").Code; code != http.StatusOK {
		t.Errorf("prefixed status = %d, want 200", code)
	}
	if code := get(t, cfg, "/health-check").Code; code != http.StatusNotFound {
		t.Errorf("unprefixed status = %d, want 404", code)
	}
}

func TestBuildInfoReportsIdentity(t *testing.T) {
	rec := get(t, Config{Commit: "abc123", BuildTime: "2026-01-01", Environment: "test"}, "/build-info.txt")
	for _, want := range []string{"commit_hash: abc123", "build_time: 2026-01-01", "environment_name: test"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("build-info missing %q, got:\n%s", want, rec.Body.String())
		}
	}
}

func TestProfilingIsOptIn(t *testing.T) {
	if code := get(t, Config{}, "/debug/pprof/").Code; code != http.StatusNotFound {
		t.Errorf("pprof status = %d with profiling off, want 404", code)
	}
	if code := get(t, Config{Profiling: true}, "/debug/pprof/").Code; code != http.StatusOK {
		t.Errorf("pprof status = %d with profiling on, want 200", code)
	}
}
