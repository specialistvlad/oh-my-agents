package tracker

import (
	"fmt"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/ids"
)

// Identifiers are distinct string types rather than bare strings so that a
// status key can never be passed where a field key belongs. They carry no
// format requirement: an adapter mints whatever its storage prefers.
type (
	// ItemID identifies one work item.
	ItemID string
	// CommentID identifies one comment.
	CommentID string
	// EventID identifies one entry in the activity feed.
	EventID string
)

// Schema identifiers. Each is minted by the system, permanent, and carries a
// readable stem taken from the name at creation plus a suffix that makes it
// unique (ADR-0009).
//
// The stem is frozen and will go stale: rename a type from "Bug" to "Defect"
// and its id keeps the old word. That is the point — a stem that followed the
// name would be a name, and addressing by it is what these exist to prevent.
// Nothing may parse an id or infer anything from one.
type (
	// TypeID addresses an [ItemType].
	TypeID string
	// FieldID addresses a [FieldDef] within a type.
	FieldID string
	// StatusID addresses a [Status] within a type.
	StatusID string
	// OptionID addresses one choice of a select field.
	OptionID string
)

// Mint builds a schema identifier from a name. The nonce is what makes it
// unique; the stem is what makes it readable (ADR-0009).
//
// The nonce is a parameter rather than generated here so a test can assert
// what a name reduces to without the answer changing every run.
func Mint(name, nonce string) string { return ids.Mint(name, "x", nonce) }

// validID reports whether an identifier could be one this system minted.
func validID(kind, id string) error {
	if err := ids.Validate(kind, id); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSchema, err)
	}
	return nil
}

// ActorKind distinguishes who acted. Agents are first-class: every place an
// actor appears accepts one, and nothing in the model treats agent activity
// as second class.
type ActorKind string

// The actor kinds.
const (
	// ActorHuman is a person acting through the UI or API.
	ActorHuman ActorKind = "human"
	// ActorAgent is an autonomous agent acting on its own behalf.
	ActorAgent ActorKind = "agent"
	// ActorSystem is the platform itself, for automation with no
	// identifiable initiator.
	ActorSystem ActorKind = "system"
)

// ActorRef points at whoever performed an action. ID is scoped by Kind, so a
// human and an agent may share an ID string without colliding.
type ActorRef struct {
	Kind ActorKind `json:"kind"`
	ID   string    `json:"id"`
}
