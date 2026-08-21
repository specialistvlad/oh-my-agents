package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Document is one setting's value: a JSON document, exactly as stored.
//
// Adapters hand back a copy, never their own memory, so a caller that
// modifies what it was given cannot corrupt the store.
type Document []byte

// Validate reports whether the document is well formed, wrapping
// [ErrInvalidDocument]. Storing anything that is not valid JSON is refused —
// see [Store] for why that is the adapter's job rather than a caller's.
func (d Document) Validate() error {
	if len(d) == 0 {
		return fmt.Errorf("%w: empty", ErrInvalidDocument)
	}
	if !json.Valid(d) {
		return fmt.Errorf("%w: not valid JSON", ErrInvalidDocument)
	}
	return nil
}

// Errors every implementation returns. An adapter translates whatever its
// backend raises into one of these and lets nothing else escape, so callers
// can branch on failure without knowing what is underneath.
var (
	// ErrNotFound is a key that holds no value.
	ErrNotFound = errors.New("settings: not found")
	// ErrInvalidKey is a key that does not satisfy [Key.Validate].
	ErrInvalidKey = errors.New("settings: invalid key")
	// ErrInvalidDocument is a value that is not a valid JSON document.
	ErrInvalidDocument = errors.New("settings: invalid document")
)

// Reader reads one setting.
type Reader interface {
	Get(ctx context.Context, key Key) (Document, error)
}

// Writer stores and removes settings.
type Writer interface {
	Set(ctx context.Context, key Key, doc Document) error
	Delete(ctx context.Context, key Key) error
}

// Lister enumerates what is stored, sorted, so a caller can page or diff
// without depending on an adapter's iteration order.
type Lister interface {
	Keys(ctx context.Context) ([]Key, error)
}

// Store composes the three ports and is what an adapter asserts against:
//
//	var _ settings.Store = (*settings.FS)(nil)
//
// It is the conformance target, not a dependency: a consumer takes the
// narrow port it needs — usually just [Reader] — so it cannot reach for
// capability it has no business having.
//
// # What an implementation guarantees
//
// Validation is the adapter's job, not a layer above it (ADR-0005). Every
// method rejects a key failing [Key.Validate] with [ErrInvalidKey], and Set
// rejects a document failing [Document.Validate] with [ErrInvalidDocument].
// A filesystem cannot enforce that for us; a database would, through a check
// constraint, and would then need none of this code.
//
// Get and Delete report [ErrNotFound] for an absent key. Set overwrites
// silently and is atomic: a reader sees the old document or the new one,
// never a partial write. Documents crossing the boundary are copies in both
// directions. Every implementation is safe for concurrent use.
//
// The suite in settingstest asserts all of the above against any
// implementation, which is what makes swapping one for another safe.
type Store interface {
	Reader
	Writer
	Lister
}
