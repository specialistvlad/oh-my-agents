package settings_test

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

type agentPrefs struct {
	Model  string `json:"model"`
	Budget int    `json:"budget"`
}

func TestWriteThenRead(t *testing.T) {
	s, ctx := settings.NewMemory(), t.Context()
	want := agentPrefs{Model: "opus", Budget: 5}

	if err := settings.Write(ctx, s, "agent/prefs", want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := settings.Read[agentPrefs](ctx, s, "agent/prefs")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != want {
		t.Errorf("Read = %+v, want %+v", got, want)
	}
}

func TestReadMissingIsNotFound(t *testing.T) {
	_, err := settings.Read[agentPrefs](t.Context(), settings.NewMemory(), "absent")
	if !errors.Is(err, settings.ErrNotFound) {
		t.Errorf("Read of absent key = %v, want ErrNotFound", err)
	}
}

func TestReadOrFallsBackOnlyOnAbsence(t *testing.T) {
	s, ctx := settings.NewMemory(), t.Context()
	fallback := agentPrefs{Model: "sonnet", Budget: 1}

	got, err := settings.ReadOr(ctx, s, "absent", fallback)
	if err != nil {
		t.Fatalf("ReadOr: %v", err)
	}
	if got != fallback {
		t.Errorf("ReadOr = %+v, want the fallback %+v", got, fallback)
	}

	// A stored document that does not fit T is a real problem, not a
	// reason to quietly hand back the default.
	if err := s.Set(ctx, "wrong-shape", settings.Document(`{"model":42}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := settings.ReadOr(ctx, s, "wrong-shape", fallback); err == nil {
		t.Error("ReadOr swallowed a decode failure; only absence may fall back")
	}
}

func TestReadOrPropagatesInvalidKey(t *testing.T) {
	_, err := settings.ReadOr(t.Context(), settings.NewMemory(), "../bad", agentPrefs{})
	if !errors.Is(err, settings.ErrInvalidKey) {
		t.Errorf("ReadOr with a bad key = %v, want ErrInvalidKey", err)
	}
}
