package tracker

import "time"

// Query selects items. Every criterion is an equality or a set membership,
// combined with AND, and there is deliberately no expression language: a
// filesystem adapter has to be able to answer this by walking files, so
// anything it could not honor does not belong here.
type Query struct {
	// Types, Statuses and Categories each match if the item matches any
	// member. Categories lets a caller ask for "everything unresolved"
	// without naming a single user-defined status.
	Types      []TypeKey
	Statuses   []StatusKey
	Categories []StatusCategory

	// Parent restricts to the direct children of one item. Roots is the
	// separate question of items with no parent at all.
	Parent *ItemID
	Roots  bool

	// Subtree restricts to every descendant of one item at any depth.
	Subtree *ItemID

	// Fields are custom-field equality matches, all of which must hold.
	Fields []FieldMatch

	// UpdatedSince is the one range criterion, because incremental readers
	// need it and every backend can answer it.
	UpdatedSince *time.Time

	Sort Sort
	Page PageRequest
}

// FieldMatch is one custom field required to equal one value.
type FieldMatch struct {
	Field  FieldKey
	Equals Value
}

// Sort orders a result set. The keys are fixed rather than open, so that
// every adapter can guarantee the same order for the same query — an
// ordering only one backend happens to provide is not an ordering.
type Sort struct {
	By   SortKey
	Desc bool
}

// SortKey names a sortable attribute.
type SortKey string

// The sort keys.
const (
	// SortCreatedAt orders by creation time. It is the default.
	SortCreatedAt SortKey = "created_at"
	// SortUpdatedAt orders by last modification.
	SortUpdatedAt SortKey = "updated_at"
	// SortTitle orders lexicographically by title.
	SortTitle SortKey = "title"
)

// PageRequest asks for one page. An empty Cursor starts at the beginning.
type PageRequest struct {
	Limit  int
	Cursor Cursor
}

// Cursor is an adapter-opaque position in a result set. Callers pass back
// what they were given and never construct one.
type Cursor string

// Page carries one page of results. Next is empty on the last page.
type Page[T any] struct {
	Rows []T
	Next Cursor
}

// EventQuery selects activity. Since is how an incremental reader resumes:
// events strictly after that sequence number, in order.
type EventQuery struct {
	Item  *ItemID
	Kinds []EventKind
	Since uint64
	Page  PageRequest
}
