package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpserver"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingshttp"
)

func main() {
	os.Exit(run())
}

// run holds main's body so deferred cleanup executes before the process
// exits with a meaningful code — os.Exit in main would skip defers.
func run() int {
	app, err := config.FromEnv()
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		return 1
	}
	initLogger(&app)
	slog.Info("starting", "port", app.HTTPPort, "env", app.Environment, "build_time", app.BuildTime)

	// The composition root: every dependency is built here and handed
	// down, so the wiring is readable in one place (ADR-0003).
	store := settings.NewFS(app.SettingsDir)
	slog.Info("settings store ready", "dir", resolve(store.Dir()))

	srv, httpErr := httpserver.Start(httpserver.Config{
		Port:        app.HTTPPort,
		Prefix:      app.APIPrefix,
		Commit:      app.CommitHash,
		BuildTime:   app.BuildTime,
		Environment: app.Environment,
		Profiling:   app.EnableProfiling,
		Timeouts:    app.Server.HTTP,
		Mounts: []httpserver.Mount{
			{Prefix: "/settings/", Handler: settingshttp.New(store)},
		},
	})
	if srv == nil {
		slog.Error("HTTP server failed to start", "err", <-httpErr)
		return 1
	}

	exitCode := 0
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	select {
	case <-quit:
		slog.Info("shutting down...")
	case err := <-httpErr:
		slog.Error("HTTP server failed, shutting down", "err", err)
		exitCode = 1
	}

	shutCtx, shutCancel := context.WithTimeout(context.Background(), app.Server.HTTP.ShutdownTimeout)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("HTTP server shutdown error", "err", err)
	}
	slog.Info("shutdown complete")
	return exitCode
}

// resolve makes a relative settings dir absolute for the boot log, so an
// operator can see exactly which directory was picked. A failure here is not
// worth failing a boot over — the relative path still says enough.
func resolve(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}
