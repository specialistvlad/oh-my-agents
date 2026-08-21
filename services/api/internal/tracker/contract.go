package tracker

// Store is not an interface — it is the contract every storage adapter
// satisfies, written down in one place because the ports themselves cannot
// express it.
//
// # Enforcement belongs to the adapter
//
// A store does not merely persist what it is handed. It guarantees the
// invariants below, and rejects any write that would break one. How it does
// that is entirely its own business, and the two obvious backends do it
// completely differently:
//
//   - A filesystem knows nothing about item types, statuses or trees. The
//     filesystem adapter therefore implements every check itself, in Go, and
//     [Validator] exists so it does not have to write the schema half by hand.
//   - A SQL database already enforces this kind of thing for a living.
//     The SQL adapter is expected to push the invariants down into the schema
//     — foreign keys, check constraints, a status-transition table, a trigger
//     for the resolution gate — and to write no validation code of its own.
//     Duplicating in Go what the database already guarantees is the mistake
//     this design exists to avoid.
//
// The invariants are the contract; the mechanism is an implementation detail.
//
// # The invariants
//
// Schema. The item's type exists. Every key in [Item.Fields] is declared by
// that type. Every value's payload matches its field's [FieldKind]. Required
// fields hold values. Select values name declared options. No field key uses
// the reserved "@" prefix.
//
// Workflow. The status belongs to the item's type. A new item enters
// [ItemType.Initial] without consulting the graph. Every later move appears
// in [ItemType.Transitions], and that transition's RequiredFields hold values
// before it is allowed.
//
// Hierarchy. The tree stays acyclic: an item may not become its own
// descendant. Depth is unbounded and any type may parent any other.
//
// Resolution. An item may not enter a resolved category — [CategoryDone] or
// [CategoryCanceled] — while any descendant sits in an unresolved one. This
// is the rule that makes the tree mean something: closing a parent asserts
// that everything under it is settled.
//
// Concurrency. Every write states the [Version] it expects and fails with
// [ErrVersionConflict] if the stored version has moved on.
//
// # Why this is safe to distribute
//
// Rules living in adapters is only replaceable if every adapter enforces them
// identically — otherwise swapping storage changes behavior, which is the
// one thing ADR-0002 forbids. A shared conformance suite is what closes that
// gap: one set of tests, written against these ports, that every adapter must
// pass. The filesystem store and the SQL store prove the same guarantees by
// different means, and the suite is what says so.
//
// The suite does not exist yet. Neither does either adapter.
//
// # Store versus the ports
//
// Store is the conformance target: what an adapter asserts against, with
// var _ tracker.Store = (*fsStore)(nil). It is wide on purpose, and it is the
// one interface in this package a consumer should never depend on — callers
// take the narrow port they need, which is the rule [SchemaReader] and the
// rest exist to serve.
type Store interface {
	SchemaReader
	SchemaWriter
	ItemReader
	ItemFinder
	ItemWriter
	SubtreeReader
	CommentReader
	CommentWriter
	LinkReader
	LinkWriter
	EventReader
}
