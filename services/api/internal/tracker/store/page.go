package store

import (
	"fmt"
	"strconv"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// defaultLimit is the page size when a caller does not ask for one. Every
// listing is paged, so an unbounded result set is never returned by accident.
const defaultLimit = 50

// paginate cuts one page out of an already-ordered slice.
//
// The cursor is an offset. That is honest for a fake and wrong for anything
// durable — an offset shifts when rows are inserted — so a real adapter is
// expected to encode a position instead. Cursors are opaque precisely so that
// each adapter can make that choice.
func paginate[T any](rows []T, req tracker.PageRequest) (tracker.Page[T], error) {
	start, err := offsetOf(req.Cursor)
	if err != nil {
		return tracker.Page[T]{}, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if start > len(rows) {
		start = len(rows)
	}
	end := min(start+limit, len(rows))

	page := tracker.Page[T]{Rows: append([]T(nil), rows[start:end]...)}
	if end < len(rows) {
		page.Next = tracker.Cursor(strconv.Itoa(end))
	}
	return page, nil
}

func offsetOf(c tracker.Cursor) (int, error) {
	if c == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(string(c))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%w: %q", tracker.ErrInvalidCursor, c)
	}
	return n, nil
}
