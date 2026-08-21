package bus_test

import (
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus/bustest"
)

func TestMemoryConformance(t *testing.T) {
	bustest.Run(t, func(t *testing.T) bus.Bus {
		b := bus.NewMemory()
		t.Cleanup(func() { _ = b.Close() })
		return b
	})
}

func TestMemoryClosedRefusesUse(t *testing.T) {
	b := bus.NewMemory()
	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := b.Publish(t.Context(), bus.Message{Rooms: []bus.Room{"r"}}); err == nil {
		t.Error("Publish succeeded on a closed bus")
	}
	if _, err := b.Subscribe(t.Context()); err == nil {
		t.Error("Subscribe succeeded on a closed bus")
	}
	if err := b.Close(); err != nil {
		t.Errorf("second Close = %v, want nil", err)
	}
}

func TestMemoryCloseEndsSubscriptions(t *testing.T) {
	b := bus.NewMemory()
	ch, err := b.Subscribe(t.Context())
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, open := <-ch; open {
		t.Error("subscription delivered after Close")
	}
}
