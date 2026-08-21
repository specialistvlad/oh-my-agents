// Package httpserver serves the API surface plus the health-check and
// build-info endpoints the deployment probes.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
)

// portProbeAttempts is how many consecutive ports Start tries when the
// requested one is busy, so several local instances can boot unattended.
const portProbeAttempts = 10

// Config holds the parameters for the HTTP server.
type Config struct {
	Port        string // listen port, no leading colon
	Prefix      string // URL prefix mounted before each route, e.g. "/api"
	Commit      string
	BuildTime   string
	Environment string
	Profiling   bool // mount /debug/pprof/* (opt-in)

	// Mounts are the application's own handlers. Keeping them a parameter
	// is what lets this package stay infrastructure: it serves whatever it
	// is handed and knows nothing about what any of it does.
	Mounts []Mount

	Timeouts config.HTTPCfg
}

// Mount attaches a handler to a subtree. Prefix is relative to Config.Prefix
// and must start and end with "/", e.g. "/settings/". The handler sees paths
// with the prefix stripped, so it registers relative patterns and does not
// care where it was mounted.
type Mount struct {
	Prefix  string
	Handler http.Handler
}

// Start binds a listener (probing upward on EADDRINUSE), serves in a
// background goroutine, and returns the server plus an error channel.
//
// The channel receives at most one value, and only when Serve fails for a
// reason other than http.ErrServerClosed. It is never closed: a closed
// channel would deliver a nil value and read as an error after a clean
// shutdown.
func Start(cfg Config) (*http.Server, <-chan error) {
	errCh := make(chan error, 1)
	ln, boundPort, err := listenWithProbe(cfg.Port)
	if err != nil {
		errCh <- err
		return nil, errCh
	}

	srv := &http.Server{
		// Serve(ln) ignores Addr; set it so callers (and tests) can read
		// the port actually bound when Port is "0" or was probed upward.
		Addr:         ln.Addr().String(),
		Handler:      newHandler(cfg),
		ReadTimeout:  cfg.Timeouts.ReadTimeout,
		WriteTimeout: cfg.Timeouts.WriteTimeout,
		IdleTimeout:  cfg.Timeouts.IdleTimeout,
	}
	go func() {
		slog.Info("HTTP server listening", "port", boundPort, "prefix", cfg.Prefix)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "err", err)
			errCh <- err
		}
	}()
	return srv, errCh
}

// listenWithProbe binds basePort, walking upward by 1 on "address already in
// use" for up to portProbeAttempts tries. Any other error returns immediately.
// The bound port is read back off the listener, so port "0" resolves to the
// arbitrary port the kernel picked.
func listenWithProbe(basePort string) (net.Listener, string, error) {
	p, err := strconv.Atoi(basePort)
	if err != nil {
		return nil, "", fmt.Errorf("invalid base port %q: %w", basePort, err)
	}
	var lc net.ListenConfig
	for i := range portProbeAttempts {
		ln, err := lc.Listen(context.Background(), "tcp", ":"+strconv.Itoa(p+i))
		if err == nil {
			bound := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
			if i > 0 {
				slog.Warn("HTTP base port busy, advanced", "requested", basePort, "bound", bound)
			}
			return ln, bound, nil
		}
		if !isAddrInUse(err) {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("no free port found in range %d-%d", p, p+portProbeAttempts-1)
}

func isAddrInUse(err error) bool {
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}
	var sysErr *os.SyscallError
	if !errors.As(opErr.Err, &sysErr) {
		return false
	}
	return errors.Is(sysErr.Err, syscall.EADDRINUSE)
}
