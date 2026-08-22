package tracker

import "time"

// Event is one entry in the append-only activity feed: what changed, who
// changed it, when. Events are the record every consumer reads to learn that
// something happened, rather than polling items for differences.
type Event struct {
	ID   EventID   `json:"id"`
	Item ItemID    `json:"item"`
	Kind EventKind `json:"kind"`
	// Seq orders events across the whole feed and never repeats. A reader
	// resumes by asking for everything after the last Seq it handled.
	Seq   uint64    `json:"seq"`
	Actor ActorRef  `json:"actor"`
	At    time.Time `json:"at"`
	// Changes is populated for edit events and empty for the rest.
	Changes []Change `json:"changes"`
}

// EventKind names what happened.
type EventKind string

// The event kinds.
const (
	// EventItemCreated is a new item.
	EventItemCreated EventKind = "item_created"
	// EventItemUpdated is a change to a title, body or custom field.
	EventItemUpdated EventKind = "item_updated"
	// EventStatusChanged is a workflow transition, called out separately
	// because it is what most consumers actually wait for.
	EventStatusChanged EventKind = "status_changed"
	// EventParentChanged is a reparenting, which moves a whole subtree.
	EventParentChanged EventKind = "parent_changed"
	// EventItemDeleted is a removed item. A reader that never sees this
	// cannot tell a deletion from an item it simply has not heard about.
	EventItemDeleted EventKind = "item_deleted"
	// EventCommentAdded is a new comment.
	EventCommentAdded EventKind = "comment_added"
	// EventCommentEdited is an edited comment.
	EventCommentEdited EventKind = "comment_edited"
	// EventCommentDeleted is a removed comment.
	EventCommentDeleted EventKind = "comment_deleted"
	// EventLinkAdded is a new link.
	EventLinkAdded EventKind = "link_added"
	// EventLinkRemoved is a removed link.
	EventLinkRemoved EventKind = "link_removed"
)

// Change is one field's before and after. Built-in parts of an item are
// described with the reserved keys — [FieldTitle], [FieldStatus] and the
// rest — so a consumer reads status changes and custom-field changes the
// same way.
//
// From is nil when the field had no value, To is nil when it was cleared.
//
// The built-in keys carry the kinds they can: [FieldTitle], [FieldBody] and
// [FieldStatus] as text, [FieldParent] as an item reference. Status is text
// rather than a kind of its own because a status key is only meaningful
// against a type, and an event is read without one.
type Change struct {
	Field FieldID `json:"field"`
	From  *Value  `json:"from"`
	To    *Value  `json:"to"`
}
