package config

import "time"

// ServerConfig groups the tunable operational parameters.
type ServerConfig struct {
	HTTP HTTPCfg
}

// HTTPCfg bounds how long the HTTP server waits on each phase of a request
// and on shutdown.
type HTTPCfg struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// AppConfig is the fully-resolved configuration a process runs with:
// deploy-time env values plus the operational defaults.
type AppConfig struct {
	Environment     string
	LogLevel        string
	HTTPPort        string
	APIPrefix       string
	EnableProfiling bool
	CommitHash      string
	BuildTime       string

	// SettingsDir is OMA_HOME verbatim, empty when unset.
	//
	// It is passed through rather than resolved here on purpose. This
	// package reads env and nothing else; the workspace owns where .oma
	// lives, what its default is, and how "~" expands. The two config
	// sources — env at boot, the .oma folder at runtime — stay independent,
	// and this string is the only thing that passes between them.
	SettingsDir string

	Server ServerConfig
}
