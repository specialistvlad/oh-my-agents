// Package feature_test exercises whole scenarios end to end against
// in-memory fakes — no infrastructure, so it runs in CI.
package feature_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpserver"
)

func TestServerServesHealthCheck(t *testing.T) {
	t.Parallel()

	srv, errCh := httpserver.Start(httpserver.Config{
		Port:     "0",
		Timeouts: config.DefaultServerConfig().HTTP,
	})
	if srv == nil {
		t.Fatalf("Start: %v", <-errCh)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	// Port "0" binds an arbitrary free port; Start records it on Addr.
	addr := srv.Addr

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/health-check", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /health-check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "OK" {
		t.Errorf("body = %q, want OK", body)
	}
}
