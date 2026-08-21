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

	Server ServerConfig
}
