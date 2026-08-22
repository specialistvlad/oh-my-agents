package feature_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpserver"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingshttp"
)

// The whole path an operator actually uses: a real server, a real store, a
// real directory on disk.
func TestSettingsSurviveThroughTheServer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := settings.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	srv, errCh := httpserver.Start(httpserver.Config{
		Port:     "0",
		Timeouts: config.DefaultServerConfig().HTTP,
		Mounts: []httpserver.Mount{
			{Prefix: "/settings/", Handler: settingshttp.New(store)},
		},
	})
	if srv == nil {
		t.Fatalf("Start: %v", <-errCh)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	base := "http://" + srv.Addr + "/settings/"

	if code, _ := request(t, http.MethodPut, base+"agent/model", `{"m":"opus"}`); code != http.StatusNoContent {
		t.Fatalf("PUT = %d, want 204", code)
	}
	code, body := request(t, http.MethodGet, base+"agent/model", "")
	if code != http.StatusOK {
		t.Fatalf("GET = %d, want 200", code)
	}
	if body != `{"m":"opus"}` {
		t.Errorf("GET body = %s, want the stored document", body)
	}

	// A second store over the same directory sees it: the setting is on
	// disk, not in the first store's memory.
	fresh, err := settings.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	got, err := settings.Read[map[string]string](t.Context(), fresh, "agent/model")
	if err != nil {
		t.Fatalf("reading through a fresh store: %v", err)
	}
	if got["m"] != "opus" {
		t.Errorf("fresh store read %v, want the persisted value", got)
	}
}

func request(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out)
}
