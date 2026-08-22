package tracker

import (
	"fmt"
	"regexp"
	"strings"
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

// minted is the shape every schema identifier takes: a readable stem, a
// hyphen, and a suffix. Deliberately narrow, so an id is always safe as both
// a path segment and a URL segment.
var minted = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// maxIDLen keeps an id inside the path limits of any filesystem it lands on.
const maxIDLen = 128

// Mint builds an identifier from a name and a nonce.
//
// A name with nothing usable in it — punctuation, or a script this reduction
// does not handle — still gets an id, because the nonce alone addresses it
// and refusing the name would be worse.
func Mint(name, nonce string) string {
	stem := stemOf(name)
	if stem == "" {
		return "x-" + nonce
	}
	return stem + "-" + nonce
}

// maxStemLen bounds the readable half, so a long name still yields a
// manageable id.
const maxStemLen = 40

var notStem = regexp.MustCompile(`[^a-z0-9]+`)

func stemOf(name string) string {
	stem := notStem.ReplaceAllString(strings.ToLower(name), "-")
	stem = strings.Trim(stem, "-")
	if len(stem) > maxStemLen {
		stem = strings.Trim(stem[:maxStemLen], "-")
	}
	return stem
}

// validID reports whether an identifier could be one this system minted.
func validID(kind, id string) error {
	switch {
	case id == "":
		return fmt.Errorf("%w: empty %s id", ErrInvalidSchema, kind)
	case len(id) > maxIDLen:
		return fmt.Errorf("%w: %s id longer than %d bytes", ErrInvalidSchema, kind, maxIDLen)
	case !minted.MatchString(id):
		return fmt.Errorf("%w: %s id %q is not one this system mints", ErrInvalidSchema, kind, id)
	default:
		return nil
	}
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
