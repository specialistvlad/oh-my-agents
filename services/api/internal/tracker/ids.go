package tracker

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

// Keys name things the schema defines. They are stable across renames: a
// status may be relabelled "In Review" without its key changing, so stored
// items and orchestration logic keep working.
type (
	// TypeKey names an [ItemType], e.g. "epic" or "bug".
	TypeKey string
	// FieldKey names a [FieldDef] within a type.
	FieldKey string
	// StatusKey names a [Status] within a type.
	StatusKey string
	// OptionKey names one choice of a select field.
	OptionKey string
)

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
	Kind ActorKind
	ID   string
}
