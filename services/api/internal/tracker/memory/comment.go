package memory

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Comments implements [tracker.CommentReader], oldest first.
func (s *Store) Comments(
	ctx context.Context, id tracker.ItemID, page tracker.PageRequest,
) (tracker.CommentPage, error) {
	if err := ctx.Err(); err != nil {
		return tracker.CommentPage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.load(id); err != nil {
		return tracker.CommentPage{}, err
	}
	var matched []tracker.Comment
	for _, c := range s.comments {
		if c.Item == id {
			matched = append(matched, cloneComment(c))
		}
	}
	// ID breaks the tie so the order is total: two comments written in the
	// same instant must still page deterministically.
	slices.SortStableFunc(matched, func(a, b tracker.Comment) int {
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(string(a.ID), string(b.ID))
	})
	return paginate(matched, page)
}

// AddComment implements [tracker.CommentWriter].
func (s *Store) AddComment(ctx context.Context, n tracker.NewComment) (tracker.Comment, error) {
	if err := ctx.Err(); err != nil {
		return tracker.Comment{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.load(n.Item); err != nil {
		return tracker.Comment{}, err
	}
	if err := s.checkReplyTo(n); err != nil {
		return tracker.Comment{}, err
	}
	now := s.clock.Now()
	c := tracker.Comment{
		ID:        tracker.CommentID(s.ids.NewID()),
		Item:      n.Item,
		Author:    n.Author,
		Body:      n.Body,
		ReplyTo:   n.ReplyTo,
		CreatedAt: now,
		Version:   1,
	}
	s.comments[c.ID] = c
	s.emit(n.Item, tracker.EventCommentAdded, n.Author, now, nil)
	return cloneComment(c), nil
}

// checkReplyTo enforces one level of threading: a reply must exist, sit on
// the same item, and not itself be a reply.
func (s *Store) checkReplyTo(n tracker.NewComment) error {
	if n.ReplyTo == nil {
		return nil
	}
	parent, ok := s.comments[*n.ReplyTo]
	if !ok {
		return fmt.Errorf("%w: comment %q", tracker.ErrNotFound, *n.ReplyTo)
	}
	if parent.Item != n.Item {
		return fmt.Errorf("%w: comment %q is on a different item", tracker.ErrNotFound, *n.ReplyTo)
	}
	if parent.ReplyTo != nil {
		return fmt.Errorf("%w: comment %q is already a reply", tracker.ErrNotFound, *n.ReplyTo)
	}
	return nil
}

// EditComment implements [tracker.CommentWriter].
func (s *Store) EditComment(
	ctx context.Context, id tracker.CommentID, expected tracker.Version, body string, by tracker.ActorRef,
) (tracker.Comment, error) {
	if err := ctx.Err(); err != nil {
		return tracker.Comment{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	c, err := s.loadComment(id, expected)
	if err != nil {
		return tracker.Comment{}, err
	}
	now := s.clock.Now()
	c.Body = body
	c.EditedAt = &now
	c.Version++
	s.comments[id] = c
	s.emit(c.Item, tracker.EventCommentEdited, by, now, nil)
	return cloneComment(c), nil
}

// DeleteComment implements [tracker.CommentWriter].
func (s *Store) DeleteComment(
	ctx context.Context, id tracker.CommentID, expected tracker.Version, by tracker.ActorRef,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	c, err := s.loadComment(id, expected)
	if err != nil {
		return err
	}
	delete(s.comments, id)
	s.emit(c.Item, tracker.EventCommentDeleted, by, s.clock.Now(), nil)
	return nil
}

func (s *Store) loadComment(id tracker.CommentID, expected tracker.Version) (tracker.Comment, error) {
	c, ok := s.comments[id]
	if !ok {
		return tracker.Comment{}, fmt.Errorf("%w: comment %q", tracker.ErrNotFound, id)
	}
	if c.Version != expected {
		return tracker.Comment{}, fmt.Errorf("%w: comment %q is at version %d, not %d",
			tracker.ErrVersionConflict, id, c.Version, expected)
	}
	return c, nil
}

// cloneComment copies the pointers a comment carries, so a caller writing
// through EditedAt or ReplyTo cannot reach the stored value.
func cloneComment(c tracker.Comment) tracker.Comment {
	out := c
	if c.ReplyTo != nil {
		replyTo := *c.ReplyTo
		out.ReplyTo = &replyTo
	}
	if c.EditedAt != nil {
		editedAt := *c.EditedAt
		out.EditedAt = &editedAt
	}
	return out
}
