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

	// Origins may read responses from a browser. Empty allows none, which
	// is correct for a service with no browser client and wrong the moment
	// there is one.
	Origins []string

	Timeouts config.HTTPCfg
}

// Mount attaches a handler at a path relative to Config.Prefix.
//
// A Prefix ending in "/" mounts a subtree — "/settings/" serves
// "/settings/anything" — and the handler sees paths with the prefix stripped,
// so it registers relative patterns and does not care where it was mounted.
// A Prefix without the trailing slash mounts that one exact path, which is
// what a single endpoint like "/ws" wants; such a handler sees an empty path
// and must not be a router.
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
	ln, boundPort, err := listen(cfg.Port)
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

// listen binds the configured port, and fails if it cannot.
//
// It does not look for another port. Quietly moving to the next one is how a
// service ends up on the port belonging to something else — the api walked
// onto the web app's port and the web app then refused to start, with nothing
// but a log line to say why. A busy port is a mistake worth stopping for, and
// the message says which port and what to do about it.
//
// Port "0" still means "any free port", which is what tests ask for and is an
// explicit request rather than a silent fallback.
func listen(port string) (net.Listener, string, error) {
	if _, err := strconv.Atoi(port); err != nil {
		return nil, "", fmt.Errorf("invalid port %q: %w", port, err)
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", ":"+port)
	if err != nil {
		if isAddrInUse(err) {
			return nil, "", fmt.Errorf(
				"port %s is already in use: stop whatever is on it, or set API_HTTP_PORT to a free one", port)
		}
		return nil, "", fmt.Errorf("cannot listen on port %s: %w", port, err)
	}
	return ln, strconv.Itoa(ln.Addr().(*net.TCPAddr).Port), nil
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
