package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FromEnv reads all deploy-time config from environment variables.
// It returns an error if any value is present but invalid.
func FromEnv() (AppConfig, error) {
	// INSTANCE_INDEX is a local-dev affordance for running several
	// instances side by side; its only effect is the port offset.
	instanceIndex := 0
	if v := os.Getenv("INSTANCE_INDEX"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 0 {
			return AppConfig{}, fmt.Errorf("config: INSTANCE_INDEX=%q: must be a non-negative integer", v)
		}
		instanceIndex = parsed
	}

	basePort, err := strconv.Atoi(getEnv("API_HTTP_PORT", "39170"))
	if err != nil {
		return AppConfig{}, fmt.Errorf("config: API_HTTP_PORT=%q: must be a valid port number", os.Getenv("API_HTTP_PORT"))
	}

	enableProfiling, err := strconv.ParseBool(getEnv("ENABLE_PPROF", "false"))
	if err != nil {
		return AppConfig{}, fmt.Errorf("config: ENABLE_PPROF=%q: must be true or false", os.Getenv("ENABLE_PPROF"))
	}

	return AppConfig{
		Environment:     getEnv("ENVIRONMENT_NAME", "local"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		HTTPPort:        strconv.Itoa(basePort + instanceIndex),
		APIPrefix:       os.Getenv("API_PREFIX"),
		EnableProfiling: enableProfiling,
		AllowedOrigins:  splitList(getEnv("ALLOWED_ORIGINS", defaultOrigins)),
		SettingsDir:     os.Getenv("OMA_HOME"),
		CommitHash:      getEnv("COMMIT_HASH", "unknown"),
		BuildTime:       getEnv("BUILD_TIME", "unknown"),
		Server:          defaults,
	}, nil
}

// defaultOrigins is where the web app runs in development. Production sets
// ALLOWED_ORIGINS explicitly; there is no wildcard default, because a
// permissive one is the kind of thing that survives to production unnoticed.
const defaultOrigins = "http://localhost:39171,http://127.0.0.1:39171"

// splitList reads a comma-separated env value, ignoring blank entries.
func splitList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
