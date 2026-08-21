package main

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/config"
)

// initLogger installs a human-readable handler locally and JSON everywhere
// else, where logs are shipped to a collector.
func initLogger(app *config.AppConfig) {
	level := slog.LevelInfo
	if app.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	if app.Environment == "local" {
		slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: "15:04:05.000",
		})))
		return
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).With(
		"service", "api", "env", app.Environment, "commit", app.CommitHash,
	))
}
