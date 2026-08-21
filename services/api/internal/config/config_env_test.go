package config

import "testing"

func TestFromEnvDefaults(t *testing.T) {
	app, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if app.HTTPPort != "39170" {
		t.Errorf("HTTPPort = %q, want 39170", app.HTTPPort)
	}
	if app.Environment != "local" {
		t.Errorf("Environment = %q, want local", app.Environment)
	}
	if app.Server.HTTP.ReadTimeout != defaults.HTTP.ReadTimeout {
		t.Errorf("HTTP.ReadTimeout = %v, want %v", app.Server.HTTP.ReadTimeout, defaults.HTTP.ReadTimeout)
	}
}

func TestInstanceIndexOffsetsPort(t *testing.T) {
	t.Setenv("API_HTTP_PORT", "9000")
	t.Setenv("INSTANCE_INDEX", "2")

	app, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if app.HTTPPort != "9002" {
		t.Errorf("HTTPPort = %q, want 9002", app.HTTPPort)
	}
}

func TestInvalidValuesRejected(t *testing.T) {
	cases := map[string][2]string{
		"negative instance index": {"INSTANCE_INDEX", "-1"},
		"non-numeric port":        {"API_HTTP_PORT", "http"},
		"non-boolean pprof":       {"ENABLE_PPROF", "maybe"},
	}
	for name, kv := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(kv[0], kv[1])
			if _, err := FromEnv(); err == nil {
				t.Fatalf("FromEnv accepted %s=%q", kv[0], kv[1])
			}
		})
	}
}
