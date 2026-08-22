package store

import (
	"context"
	"fmt"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Links implements [tracker.LinkReader], returning links in either direction.
func (s *Store) Links(ctx context.Context, id tracker.ItemID) ([]tracker.Link, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.load(id); err != nil {
		return nil, err
	}
	var out []tracker.Link
	for _, l := range s.links {
		if l.From == id || l.To == id {
			out = append(out, l)
		}
	}
	return out, nil
}

// AddLink implements [tracker.LinkWriter]. Both ends must exist, an item
// cannot link to itself, and adding a link twice is a no-op rather than an
// error — the caller's intent is already satisfied.
func (s *Store) AddLink(ctx context.Context, l tracker.Link) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if l.From == l.To {
		return fmt.Errorf("%w: %q cannot link to itself", tracker.ErrCycle, l.From)
	}
	for _, end := range []tracker.ItemID{l.From, l.To} {
		if _, err := s.load(end); err != nil {
			return err
		}
	}
	if s.indexOfLink(l) >= 0 {
		return nil
	}
	l.CreatedAt = s.clock.Now()
	next := append(append([]tracker.Link(nil), s.links...), l)
	if err := s.disk.SaveLinks(ctx, next); err != nil {
		return err
	}
	s.links = next
	s.emit(ctx, l.From, tracker.EventLinkAdded, l.CreatedBy, l.CreatedAt, nil)
	return nil
}

// RemoveLink implements [tracker.LinkWriter].
func (s *Store) RemoveLink(ctx context.Context, l tracker.Link, by tracker.ActorRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	at := s.indexOfLink(l)
	if at < 0 {
		return fmt.Errorf("%w: link %q %s %q", tracker.ErrNotFound, l.From, l.Kind, l.To)
	}
	removed := s.links[at]
	next := append(append([]tracker.Link(nil), s.links[:at]...), s.links[at+1:]...)
	if err := s.disk.SaveLinks(ctx, next); err != nil {
		return err
	}
	s.links = next
	s.emit(ctx, removed.From, tracker.EventLinkRemoved, by, s.clock.Now(), nil)
	return nil
}

// indexOfLink finds a link by its three identifying parts; who created it and
// when are not part of its identity.
func (s *Store) indexOfLink(l tracker.Link) int {
	for i, existing := range s.links {
		if existing.From == l.From && existing.To == l.To && existing.Kind == l.Kind {
			return i
		}
	}
	return -1
}
