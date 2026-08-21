// Package tracker is the domain model for the task management system:
// the work items agents and people collaborate on, and the ports through
// which they are stored and read.
//
// The model has three layers.
//
// Schema is configuration, editable at runtime. An [ItemType] — epic,
// feature, user story, requirement, release, bug — owns its own field set
// and its own status set. Adding a type is a write, not a deploy.
//
// Content is the data. An [Item] carries a type, a status drawn from that
// type's statuses, and values for that type's fields. [Comment], [Link] and
// [Event] hang off items.
//
// Actors are whoever acted. An [ActorRef] is a human, an agent or the system
// itself, and the three are interchangeable everywhere authorship appears —
// an agent comments and transitions work exactly as a person does.
//
// # Hierarchy
//
// Items form a tree of unbounded depth. Any item may parent any other,
// whatever their types, and there are no fixed levels — the epic/story/
// subtask ladder of other trackers is not modeled, because an agent
// decomposing work does not know in advance how deep it needs to go.
//
// The tree carries one rule: an item may not be resolved while any of its
// descendants is unresolved. Resolution is read off [StatusCategory], not
// off status names, so the rule holds no matter what a type calls its
// columns. See [SubtreeReader].
//
// This package declares types and ports only. Storage adapters, schema
// validation, transition checking and the resolution gate are not
// implemented here; see docs/spec/tracker.md for what is specified and what
// is still open.
package tracker
