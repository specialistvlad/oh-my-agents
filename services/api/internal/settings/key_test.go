package settings_test

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

func TestKeyAccepts(t *testing.T) {
	for _, k := range []settings.Key{
		"a", "agent", "agent.model", "agent/model", "a/b/c/d/e",
		"with_underscore", "with-hyphen", "MixedCase", "v2.1/limits",
	} {
		if err := settings.Key(k).Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", k, err)
		}
	}
}

func TestKeyRejects(t *testing.T) {
	for name, k := range map[string]settings.Key{
		"empty":            "",
		"dot":              ".",
		"dotdot":           "..",
		"traversal":        "../escape",
		"buried traversal": "a/../../etc/passwd",
		"absolute":         "/etc/passwd",
		"trailing slash":   "a/",
		"empty segment":    "a//b",
		"leading dot":      ".hidden",
		"trailing dot":     "a.",
		"double dot":       "a..b",
		"space":            "a b",
		"glob":             "a*",
		"backslash":        `a\b`,
		"null byte":        "a\x00b",
	} {
		t.Run(name, func(t *testing.T) {
			err := settings.Key(k).Validate()
			if !errors.Is(err, settings.ErrInvalidKey) {
				t.Errorf("Validate(%q) = %v, want ErrInvalidKey", k, err)
			}
		})
	}
}

func TestKeyRejectsOverlyLong(t *testing.T) {
	long := settings.Key(make([]byte, 0, 600))
	for range 600 {
		long += "a"
	}
	if err := long.Validate(); !errors.Is(err, settings.ErrInvalidKey) {
		t.Errorf("Validate(600 chars) = %v, want ErrInvalidKey", err)
	}
}
