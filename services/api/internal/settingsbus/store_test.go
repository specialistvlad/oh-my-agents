package settingsbus_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings/settingstest"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingsbus"
)

// Announcing must not change what a store is, so it faces the same suite
// every settings store faces.
func TestConformance(t *testing.T) {
	settingstest.Run(t, func(t *testing.T) settings.Store {
		b := bus.NewMemory()
		t.Cleanup(func() { _ = b.Close() })
		return settingsbus.New(settings.NewMemory(), b, "project:test")
	})
}

func listening(t *testing.T) (settings.Store, <-chan bus.Message) {
	t.Helper()
	b := bus.NewMemory()
	t.Cleanup(func() { _ = b.Close() })
	messages, err := b.Subscribe(t.Context())
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	return settingsbus.New(settings.NewMemory(), b, "project:test"), messages
}

func next(t *testing.T, messages <-chan bus.Message) bus.Message {
	t.Helper()
	select {
	case m := <-messages:
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("no message was published")
		return bus.Message{}
	}
}

func TestAnnouncesWrites(t *testing.T) {
	store, messages := listening(t)

	if err := store.Set(t.Context(), "agent/model", settings.Document(`{"m":"opus"}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	m := next(t, messages)
	if m.Kind != settingsbus.KindChanged || len(m.Rooms) != 1 || m.Rooms[0] != "project:test" {
		t.Errorf("message = %+v, want a setting.changed in the settings room", m)
	}

	if err := store.Delete(t.Context(), "agent/model"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if m := next(t, messages); m.Kind != settingsbus.KindDeleted {
		t.Errorf("Kind = %q, want setting.deleted", m.Kind)
	}
}

// Settings hold credentials. A payload on the bus reaches every connected
// client and the log, so only the key travels.
func TestAnnouncesTheKeyAndNotTheValue(t *testing.T) {
	store, messages := listening(t)
	secret := `{"token":"super-secret-value"}`

	if err := store.Set(t.Context(), "creds", settings.Document(secret)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	m := next(t, messages)
	if string(m.Data) == secret {
		t.Fatal("the stored value was published")
	}
	var payload struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		t.Fatalf("decode %s: %v", m.Data, err)
	}
	if payload.Key != "creds" {
		t.Errorf("Key = %q, want creds", payload.Key)
	}
}

// A rejected write changed nothing, so announcing it would be a lie.
func TestSaysNothingWhenTheWriteFailed(t *testing.T) {
	store, messages := listening(t)

	if err := store.Set(t.Context(), "../escape", settings.Document(`{}`)); !errors.Is(err, settings.ErrInvalidKey) {
		t.Fatalf("Set = %v, want ErrInvalidKey", err)
	}
	if err := store.Delete(t.Context(), "absent"); !errors.Is(err, settings.ErrNotFound) {
		t.Fatalf("Delete = %v, want ErrNotFound", err)
	}
	select {
	case m := <-messages:
		t.Errorf("a failed write was announced: %+v", m)
	case <-time.After(100 * time.Millisecond):
	}
}

// A process with no realtime surface wires the same way.
func TestWorksWithNoPublisher(t *testing.T) {
	store := settingsbus.New(settings.NewMemory(), nil, "project:test")
	if err := store.Set(context.Background(), "k", settings.Document(`{}`)); err != nil {
		t.Errorf("Set with no publisher: %v", err)
	}
}

// A publisher that fails must not fail a write that already succeeded.
func TestAWriteSurvivesAFailedAnnouncement(t *testing.T) {
	b := bus.NewMemory()
	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	store := settingsbus.New(settings.NewMemory(), b, "project:test")

	if err := store.Set(t.Context(), "k", settings.Document(`{"v":1}`)); err != nil {
		t.Fatalf("Set = %v; a durable write must not fail because nobody heard about it", err)
	}
	if _, err := store.Get(t.Context(), "k"); err != nil {
		t.Errorf("Get = %v, want the value that was written", err)
	}
}
