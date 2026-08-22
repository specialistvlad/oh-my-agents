package store

import (
	"context"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Persistence is where a store's state lives between processes.
//
// The core holds the rules; this holds the bytes. Splitting them is what lets
// a durable store exist without a second copy of the resolution gate, the
// cycle check and the version comparison — the alternative being to write all
// of that twice and hope the two agree (ADR-0001).
//
// It is wide, like [tracker.Store] and for the same reason: it is what an
// implementation asserts against, not something a consumer depends on.
//
// Every write persists **before** the change is applied in memory, so a
// failure leaves the store exactly as it was rather than holding a change
// nobody recorded.
type Persistence interface {
	// Load returns everything that was stored. An empty snapshot is a
	// perfectly good answer for somewhere nothing has been written yet.
	Load(ctx context.Context) (Snapshot, error)

	SaveType(ctx context.Context, t tracker.ItemType) error
	DeleteType(ctx context.Context, id tracker.TypeID) error

	SaveItem(ctx context.Context, item tracker.Item) error
	DeleteItem(ctx context.Context, id tracker.ItemID) error

	SaveComment(ctx context.Context, c tracker.Comment) error
	DeleteComment(ctx context.Context, id tracker.CommentID) error

	// SaveLinks writes the whole set, because a link has no identity beyond
	// its three fields and the set is small.
	SaveLinks(ctx context.Context, links []tracker.Link) error

	AppendEvent(ctx context.Context, e tracker.Event) error
}

// Snapshot is everything a store holds.
type Snapshot struct {
	Schema   tracker.Schema
	Items    []tracker.Item
	Comments []tracker.Comment
	Links    []tracker.Link
	Events   []tracker.Event
}

// nothing is the persistence of a store that keeps nothing: every write
// succeeds and is forgotten when the process ends.
type nothing struct{}

func (nothing) Load(context.Context) (Snapshot, error)                 { return Snapshot{}, nil }
func (nothing) SaveType(context.Context, tracker.ItemType) error       { return nil }
func (nothing) DeleteType(context.Context, tracker.TypeID) error       { return nil }
func (nothing) SaveItem(context.Context, tracker.Item) error           { return nil }
func (nothing) DeleteItem(context.Context, tracker.ItemID) error       { return nil }
func (nothing) SaveComment(context.Context, tracker.Comment) error     { return nil }
func (nothing) DeleteComment(context.Context, tracker.CommentID) error { return nil }
func (nothing) SaveLinks(context.Context, []tracker.Link) error        { return nil }
func (nothing) AppendEvent(context.Context, tracker.Event) error       { return nil }
