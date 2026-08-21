package tracker

import (
	"context"
	"time"
)

// The ports are deliberately small and single-purpose. A consumer depends on
// the one or two it actually uses — a reporting path takes an [ItemFinder]
// and nothing else — so it cannot reach for capability it has no business
// having, and a test fake stays a few lines long.
//
// Adapters satisfy the lot; see [Store].

// Pages returned by the finders.
type (
	// ItemPage is one page of items.
	ItemPage = Page[Item]
	// CommentPage is one page of comments.
	CommentPage = Page[Comment]
	// EventPage is one page of activity.
	EventPage = Page[Event]
)

// SchemaReader reads the type configuration.
type SchemaReader interface {
	Schema(ctx context.Context) (Schema, error)
}

// SchemaWriter edits the type configuration. A type whose own definition is
// inconsistent is rejected with [ErrInvalidSchema]; a change that would
// invalidate items already stored is rejected too.
type SchemaWriter interface {
	PutItemType(ctx context.Context, t ItemType) error
	DeleteItemType(ctx context.Context, key TypeKey) error
}

// ItemReader reads one item by identity.
type ItemReader interface {
	Item(ctx context.Context, id ItemID) (Item, error)
}

// ItemFinder searches items.
type ItemFinder interface {
	FindItems(ctx context.Context, q Query) (ItemPage, error)
}

// ItemWriter creates, edits and removes items. Every mutation states the
// version it expects and fails with [ErrVersionConflict] if the item has
// moved on, so concurrent agents cannot overwrite one another unnoticed.
type ItemWriter interface {
	CreateItem(ctx context.Context, n NewItem) (Item, error)
	UpdateItem(ctx context.Context, id ItemID, expected Version, p Patch) (Item, error)
	DeleteItem(ctx context.Context, id ItemID, expected Version) error
}

// SubtreeReader answers questions about an item's place in the tree without
// the caller walking it a node at a time.
//
// UnresolvedDescendants is what the resolution gate is built on: zero means
// everything beneath this item is settled and it may close. Ancestors, in
// root-first order, is what a reparenting checks against to refuse a cycle.
type SubtreeReader interface {
	UnresolvedDescendants(ctx context.Context, id ItemID) (int, error)
	Ancestors(ctx context.Context, id ItemID) ([]Item, error)
}

// CommentReader reads an item's comments, oldest first.
type CommentReader interface {
	Comments(ctx context.Context, id ItemID, page PageRequest) (CommentPage, error)
}

// CommentWriter posts, edits and removes comments.
type CommentWriter interface {
	AddComment(ctx context.Context, n NewComment) (Comment, error)
	EditComment(ctx context.Context, id CommentID, expected Version, body string) (Comment, error)
	DeleteComment(ctx context.Context, id CommentID, expected Version) error
}

// LinkReader reads every link touching an item, in either direction.
type LinkReader interface {
	Links(ctx context.Context, id ItemID) ([]Link, error)
}

// LinkWriter adds and removes links.
type LinkWriter interface {
	AddLink(ctx context.Context, l Link) error
	RemoveLink(ctx context.Context, l Link) error
}

// EventReader reads the activity feed. It is the seam anything reactive is
// built on: a reader resumes from the last [Event.Seq] it handled rather than
// diffing items.
type EventReader interface {
	Events(ctx context.Context, q EventQuery) (EventPage, error)
}

// Clock and IDs are the two pieces of ambient state an adapter needs and must
// not reach for directly, because tests need both to be boring.
type (
	// Clock reports the current time.
	Clock interface {
		Now() time.Time
	}
	// IDGenerator mints identifiers. Format is the adapter's business.
	IDGenerator interface {
		NewID() string
	}
)
