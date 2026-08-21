package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Read fetches a setting and decodes it into T.
//
// It is a free function rather than a method so that the port stays three
// methods wide however many types callers keep in the store — the typed
// ergonomics compose on top of the byte interface instead of widening it.
func Read[T any](ctx context.Context, r Reader, key Key) (T, error) {
	var v T
	doc, err := r.Get(ctx, key)
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(doc, &v); err != nil {
		return v, fmt.Errorf("settings: decode %q: %w", key, err)
	}
	return v, nil
}

// ReadOr fetches a setting and decodes it into T, returning fallback when the
// key is absent. Any other failure — an unreadable store, a document that
// does not fit T — is still an error, because silently falling back on those
// hides a real problem.
func ReadOr[T any](ctx context.Context, r Reader, key Key, fallback T) (T, error) {
	v, err := Read[T](ctx, r, key)
	if isNotFound(err) {
		return fallback, nil
	}
	return v, err
}

// Write encodes v and stores it.
func Write[T any](ctx context.Context, w Writer, key Key, v T) error {
	doc, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("settings: encode %q: %w", key, err)
	}
	return w.Set(ctx, key, doc)
}

// isNotFound keeps the errors.Is call in one place.
func isNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
