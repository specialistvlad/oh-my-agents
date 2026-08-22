package httpserver_test

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpserver"
)

// A busy port must stop the process, not send it looking for another one.
// Quietly moving is how the api ended up on the web app's port.
func TestABusyPortFailsLoudly(t *testing.T) {
	var lc net.ListenConfig
	held, err := lc.Listen(t.Context(), "tcp", ":0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = held.Close() })
	taken := strconv.Itoa(held.Addr().(*net.TCPAddr).Port)

	srv, errCh := httpserver.Start(httpserver.Config{
		Port:     taken,
		Timeouts: config.DefaultServerConfig().HTTP,
	})
	if srv != nil {
		t.Fatal("Start succeeded on a port already in use")
	}
	select {
	case err := <-errCh:
		if !strings.Contains(err.Error(), taken) {
			t.Errorf("error = %q, want it to name the port %s", err, taken)
		}
		if !strings.Contains(err.Error(), "already in use") {
			t.Errorf("error = %q, want it to say the port is in use", err)
		}
		if !strings.Contains(err.Error(), "API_HTTP_PORT") {
			t.Errorf("error = %q, want it to say what to do about it", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start reported no error for a busy port")
	}
}

// Port "0" is an explicit request for any free port, which tests rely on.
func TestPortZeroStillBindsAnything(t *testing.T) {
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
	if srv.Addr == "" || strings.HasSuffix(srv.Addr, ":0") {
		t.Errorf("Addr = %q, want a real bound port", srv.Addr)
	}
}
