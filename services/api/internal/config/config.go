// Package config is the single place deploy-time configuration is read and
// operational parameters are defined. Production code calls FromEnv; tests
// construct the structs directly or start from DefaultServerConfig.
package config

import "time"

// defaults holds every tunable operational parameter.
var defaults = ServerConfig{
	HTTP: HTTPCfg{
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 15 * time.Second,
	},
}

// DefaultServerConfig returns a copy of the default operational parameters.
func DefaultServerConfig() ServerConfig {
	return defaults
}
