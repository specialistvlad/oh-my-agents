package memory_test

import (
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/memory"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/trackertest"
)

func TestConformance(t *testing.T) {
	trackertest.Run(t, func(_ *testing.T) tracker.Store {
		return memory.New(memory.Deps{Clock: &stepClock{}, IDs: &countingIDs{}})
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
