package store_test

import (
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/store"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/trackertest"
)

// The core with nothing behind it: the fake every other test builds on.
func TestConformance(t *testing.T) {
	trackertest.Run(t, func(t *testing.T) tracker.Store {
		s, err := store.New(t.Context(), store.Deps{Clock: &stepClock{}, IDs: &countingIDs{}})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		return s
	})
}

// stepClock advances a fixed amount per reading, so ordering by time is
// deterministic without any test having to sleep.
type stepClock struct{ n int }

func (c *stepClock) Now() time.Time {
	c.n++
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(c.n) * time.Second)
}

// countingIDs mints predictable identifiers. Nothing may depend on the
// format, which is exactly why a test is free to make it boring.
type countingIDs struct{ n int }

func (g *countingIDs) NewID() string {
	g.n++
	return string(rune('a'+(g.n-1)%26)) + string(rune('0'+(g.n-1)/26))
}
