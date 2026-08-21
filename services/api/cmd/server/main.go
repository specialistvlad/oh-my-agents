package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/httpserver"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
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
	store, err := settings.NewFS(app.SettingsDir)
	if err != nil {
		slog.Error("cannot locate the workspace", "err", err)
		return 1
	}
	slog.Info("workspace ready", "dir", store.Dir())

	// The bus is in-memory until VALKEY_URL says otherwise (ADR-0008), so
	// this runs with nothing installed.
	messages := bus.NewMemory()
	defer func() { _ = messages.Close() }()

	hub := realtime.New()
	ctx, stopHub := context.WithCancel(context.Background())
	defer stopHub()
	pump, err := hub.Attach(ctx, messages)
	if err != nil {
		slog.Error("cannot attach the realtime hub", "err", err)
		return 1
	}
	go func() { _ = pump() }()
	slog.Info("realtime ready", "bus", "memory", "origins", app.AllowedOrigins)

	srv, httpErr := httpserver.Start(httpserver.Config{
		Port:        app.HTTPPort,
		Prefix:      app.APIPrefix,
		Commit:      app.CommitHash,
		BuildTime:   app.BuildTime,
		Environment: app.Environment,
		Profiling:   app.EnableProfiling,
		Timeouts:    app.Server.HTTP,
		Mounts: []httpserver.Mount{
			{Prefix: "/settings/", Handler: settingshttp.New(store, settingshttp.BusAnnouncer{Bus: messages})},
			{Prefix: "/ws", Handler: realtimews.New(hub, realtimews.Options{Origins: app.AllowedOrigins})},
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
