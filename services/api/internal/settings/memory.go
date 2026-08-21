package settings

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
)

// Memory keeps settings in a map. It is the fake every test builds on, and
// it exists for the reason ADR-0002 gives: a seam with only one
// implementation is a claim, not a fact.
//
// It is held to the same guarantees as [FS] by the settingstest suite.
type Memory struct {
	mu   sync.RWMutex
	docs map[Key]Document
}

// NewMemory returns an empty store.
func NewMemory() *Memory {
	return &Memory{docs: make(map[Key]Document)}
}

// Get implements [Reader].
func (s *Memory) Get(ctx context.Context, key Key) (Document, error) {
	if err := check(ctx, key); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.docs[key]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, key)
	}
	return slices.Clone(doc), nil
}

// Set implements [Writer].
func (s *Memory) Set(ctx context.Context, key Key, doc Document) error {
	if err := check(ctx, key); err != nil {
		return err
	}
	if err := doc.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.docs[key] = slices.Clone(doc)
	return nil
}

// Delete implements [Writer].
func (s *Memory) Delete(ctx context.Context, key Key) error {
	if err := check(ctx, key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.docs[key]; !ok {
		return fmt.Errorf("%w: %q", ErrNotFound, key)
	}
	delete(s.docs, key)
	return nil
}

// Keys implements [Lister].
func (s *Memory) Keys(ctx context.Context) ([]Key, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Sorted(maps.Keys(s.docs)), nil
}

// check is the entry guard both stores share: a dead context and a malformed
// key are refused before any work happens.
func check(ctx context.Context, key Key) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return key.Validate()
}
