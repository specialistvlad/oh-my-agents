package tracker

import "errors"

// Storage errors. An adapter returns these; it does not invent its own, and
// it never lets a driver error escape — translating at its own boundary is
// part of what makes it replaceable.
var (
	// ErrNotFound is any addressed thing that does not exist.
	ErrNotFound = errors.New("tracker: not found")
	// ErrVersionConflict is a write whose expected version is stale.
	// The caller re-reads and retries.
	ErrVersionConflict = errors.New("tracker: version conflict")
	// ErrInvalidCursor is a page cursor an adapter cannot interpret.
	ErrInvalidCursor = errors.New("tracker: invalid cursor")
)

// Schema errors. Raised by the validating layer, never by storage.
var (
	// ErrUnknownType names a type the schema does not define.
	ErrUnknownType = errors.New("tracker: unknown item type")
	// ErrUnknownField names a field the item's type does not declare.
	ErrUnknownField = errors.New("tracker: unknown field")
	// ErrUnknownStatus names a status the item's type does not declare.
	ErrUnknownStatus = errors.New("tracker: unknown status")
	// ErrUnknownOption is a select value that is not a declared option.
	ErrUnknownOption = errors.New("tracker: unknown option")
	// ErrFieldRequired is a required field with no value.
	ErrFieldRequired = errors.New("tracker: field required")
	// ErrKindMismatch is a value whose payload contradicts its field's kind.
	ErrKindMismatch = errors.New("tracker: value does not match field kind")
	// ErrReservedKey is a field key using the reserved "@" prefix.
	ErrReservedKey = errors.New("tracker: reserved field key")
	// ErrInvalidSchema is a type whose own definition is inconsistent, e.g.
	// an Initial status it does not declare, or a transition to nowhere.
	ErrInvalidSchema = errors.New("tracker: invalid schema")
)

// Structural errors. Raised by the validating layer, never by storage.
var (
	// ErrTransitionNotAllowed is a status move absent from the type's
	// transition graph.
	ErrTransitionNotAllowed = errors.New("tracker: transition not allowed")
	// ErrUnresolvedDescendants is an attempt to resolve an item while work
	// beneath it is still open.
	ErrUnresolvedDescendants = errors.New("tracker: item has unresolved descendants")
	// ErrCycle is a reparenting that would make an item its own ancestor.
	ErrCycle = errors.New("tracker: parent cycle")
)
