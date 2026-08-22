package tracker_test

import (
	"sort"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func between(t *testing.T, prev, next tracker.Rank) tracker.Rank {
	t.Helper()
	r, err := tracker.Between(prev, next)
	if err != nil {
		t.Fatalf("Between(%q, %q): %v", prev, next, err)
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("Between(%q, %q) = %q, which is not a valid rank: %v", prev, next, r, err)
	}
	return r
}

// The only thing a rank promises is where it sorts.
func TestBetweenLandsBetween(t *testing.T) {
	first := between(t, "", "")
	before := between(t, "", first)
	after := between(t, first, "")

	if !ordered(before, first, after) {
		t.Errorf("got %q < %q < %q, which is not in order", before, first, after)
	}
	if middle := between(t, before, after); !ordered(before, middle, after) {
		t.Errorf("%q is not between %q and %q", middle, before, after)
	}
}

// Room always exists between two neighbors, however tight. This is what lets
// a sparse key avoid renumbering siblings.
func TestThereIsAlwaysRoom(t *testing.T) {
	lo, hi := between(t, "", ""), between(t, between(t, "", ""), "")
	for range 200 {
		mid := between(t, lo, hi)
		if !ordered(lo, mid, hi) {
			t.Fatalf("%q is not between %q and %q", mid, lo, hi)
		}
		hi = mid // squeeze against the same neighbor, the worst case
	}
}

// Inserting repeatedly at one spot lengthens the key rather than failing.
// ADR-0013 accepts that growth, and what makes it acceptable is that it is
// sub-linear: each character of key absorbs several insertions, so the worst
// case a person can produce by dragging to the same spot stays short enough to
// read in a file name.
func TestKeysGrowSublinearly(t *testing.T) {
	const insertions = 100
	lo, hi := between(t, "", ""), between(t, between(t, "", ""), "")
	for range insertions {
		hi = between(t, lo, hi) // squeeze against the same neighbor every time
	}
	if len(hi) >= insertions/2 {
		t.Errorf("%d insertions at one spot grew the key to %d characters (%q); growth is not sub-linear",
			insertions, len(hi), hi)
	}
	t.Logf("worst case: %d insertions -> %d characters", insertions, len(hi))
}

// A list built by appending must read back in the order it was built.
func TestAppendingKeepsOrder(t *testing.T) {
	var ranks []tracker.Rank
	last := tracker.Rank("")
	for range 50 {
		last = between(t, last, "")
		ranks = append(ranks, last)
	}
	if !sort.SliceIsSorted(ranks, func(i, j int) bool { return ranks[i] < ranks[j] }) {
		t.Errorf("appended ranks are not sorted: %v", ranks)
	}
}

func TestBetweenRefusesNeighboursOutOfOrder(t *testing.T) {
	lo, hi := between(t, "", ""), between(t, between(t, "", ""), "")
	if _, err := tracker.Between(hi, lo); err == nil {
		t.Error("Between accepted neighbors in the wrong order")
	}
	if _, err := tracker.Between(lo, lo); err == nil {
		t.Error("Between accepted a rank between something and itself")
	}
}

func TestValidate(t *testing.T) {
	for name, r := range map[string]tracker.Rank{
		"empty":            "",
		"trailing minimum": "na",
		"capitals":         "N",
		"digits":           "n1",
		"punctuation":      "n-n",
	} {
		t.Run(name, func(t *testing.T) {
			if err := r.Validate(); err == nil {
				t.Errorf("Validate accepted %q", r)
			}
		})
	}
	for _, r := range []tracker.Rank{"n", "nn", "abz", "z"} {
		if err := r.Validate(); err != nil {
			t.Errorf("Validate refused %q: %v", r, err)
		}
	}
}

// ordered reports whether the three sort in the order given. Named because
// "is b between a and c" is the whole claim these tests make, and spelling it
// out at each site reads worse than saying it once.
func ordered(a, b, c tracker.Rank) bool {
	return a < b && b < c
}
