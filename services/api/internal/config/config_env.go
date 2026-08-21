package config

import (
	"fmt"
	"os"
	"strconv"
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
		SettingsDir:     os.Getenv("OMA_HOME"),
		CommitHash:      getEnv("COMMIT_HASH", "unknown"),
		BuildTime:       getEnv("BUILD_TIME", "unknown"),
		Server:          defaults,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
