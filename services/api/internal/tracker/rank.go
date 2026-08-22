package tracker

import (
	"fmt"
	"strings"
)

// Rank orders items within a project (ADR-0013).
//
// It is a sparse key rather than a position: inserting between two neighbors
// mints a key between theirs and writes one item, where a dense position would
// renumber every sibling. That matters because the filesystem store keeps one
// file per item (ADR-0002).
//
// Ranks sort as plain strings. Nothing may read meaning into one beyond its
// order — the alphabet and the midpoint rule are this package's business.
type Rank string

// The alphabet ranks are written in. Lowercase letters only: a rank shows up
// in stored files and log lines, and one that is readable at a glance is worth
// more than one that is two characters shorter.
const (
	rankMin  = 'a'
	rankMax  = 'z'
	rankBase = rankMax - rankMin + 1
)

// Between mints a rank that sorts after prev and before next.
//
// An empty prev means "before everything" and an empty next means "after
// everything", so the first item in a project is Between("", "") and appending
// is Between(last, "").
func Between(prev, next Rank) (Rank, error) {
	if next != "" && prev >= next {
		return "", fmt.Errorf("tracker: cannot rank between %q and %q, which are out of order", prev, next)
	}
	var out strings.Builder
	for i := 0; ; i++ {
		lo, hi := bounds(prev, next, i)
		mid := (lo + hi) / 2
		if mid == lo {
			// The two are adjacent at this position, so there is no room
			// here. Keep prev's digit and look for room one level deeper.
			out.WriteByte(byte(rankMin + lo))
			continue
		}
		out.WriteByte(byte(rankMin + mid))
		return Rank(out.String()), nil
	}
}

// bounds gives the range a digit may take at position i.
//
// Past the end of prev there is no lower constraint, so the floor is the first
// letter. Past the end of next there is no upper constraint, so the ceiling is
// one past the last — which is what lets a rank grow longer rather than run
// out of room.
func bounds(prev, next Rank, i int) (lo, hi int) {
	lo = 0
	if i < len(prev) {
		lo = int(prev[i] - rankMin)
	}
	hi = rankBase
	if i < len(next) {
		hi = int(next[i] - rankMin)
	}
	return lo, hi
}

// Validate reports whether a rank is one this package could have minted.
//
// A trailing first-letter digit is refused: it adds nothing to the order and
// means two different strings sort identically, so ranks would stop being
// comparable for equality.
func (r Rank) Validate() error {
	switch {
	case r == "":
		return fmt.Errorf("%w: empty rank", ErrInvalidSchema)
	case r[len(r)-1] == rankMin:
		return fmt.Errorf("%w: rank %q ends in %q, which adds no order", ErrInvalidSchema, r, string(rankMin))
	}
	for i := range r {
		if r[i] < rankMin || r[i] > rankMax {
			return fmt.Errorf("%w: rank %q is not in the rank alphabet", ErrInvalidSchema, r)
		}
	}
	return nil
}
