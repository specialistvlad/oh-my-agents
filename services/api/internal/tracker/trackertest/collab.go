package trackertest

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func human(id string) tracker.ActorRef {
	return tracker.ActorRef{Kind: tracker.ActorHuman, ID: id}
}

func agent(id string) tracker.ActorRef {
	return tracker.ActorRef{Kind: tracker.ActorAgent, ID: id}
}

func runComments(t *testing.T, newStore Factory) {
	t.Run("posts and reads back", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if _, err := s.AddComment(ctx, tracker.NewComment{
			Item: item.ID, Author: human("vk"), Body: "looks wrong",
		}); err != nil {
			t.Fatalf("AddComment: %v", err)
		}
		page, err := s.Comments(ctx, item.ID, tracker.PageRequest{})
		if err != nil {
			t.Fatalf("Comments: %v", err)
		}
		if len(page.Rows) != 1 || page.Rows[0].Body != "looks wrong" {
			t.Errorf("Comments = %+v, want the comment as posted", page.Rows)
		}
	})

	// Humans and agents post through the same path; an agent narrating its
	// progress is an ordinary comment, not a separate log.
	t.Run("accepts agents as authors", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		c, err := s.AddComment(ctx, tracker.NewComment{
			Item: item.ID, Author: agent("builder-1"), Body: "starting work",
		})
		if err != nil {
			t.Fatalf("AddComment: %v", err)
		}
		if c.Author.Kind != tracker.ActorAgent {
			t.Errorf("Author.Kind = %q, want agent", c.Author.Kind)
		}
	})

	t.Run("threads one level deep", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		root, err := s.AddComment(ctx, tracker.NewComment{Item: item.ID, Body: "a"})
		if err != nil {
			t.Fatalf("AddComment: %v", err)
		}
		reply, err := s.AddComment(ctx, tracker.NewComment{
			Item: item.ID, Body: "b", ReplyTo: &root.ID,
		})
		if err != nil {
			t.Fatalf("reply: %v", err)
		}
		if _, err := s.AddComment(ctx, tracker.NewComment{
			Item: item.ID, Body: "c", ReplyTo: &reply.ID,
		}); err == nil {
			t.Error("a reply to a reply was accepted; threading is one level")
		}
	})

	t.Run("edits and deletes with a version", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		c, err := s.AddComment(ctx, tracker.NewComment{Item: item.ID, Body: "typo"})
		if err != nil {
			t.Fatalf("AddComment: %v", err)
		}
		edited, err := s.EditComment(ctx, c.ID, c.Version, "fixed", human("moderator"))
		if err != nil {
			t.Fatalf("EditComment: %v", err)
		}
		if edited.Body != "fixed" || edited.EditedAt == nil {
			t.Errorf("EditComment = %+v, want the new body and an edit stamp", edited)
		}
		if _, err := s.EditComment(ctx, c.ID, c.Version, "again", human("moderator")); !errors.Is(err, tracker.ErrVersionConflict) {
			t.Errorf("stale edit = %v, want ErrVersionConflict", err)
		}
		if err := s.DeleteComment(ctx, edited.ID, edited.Version, human("moderator")); err != nil {
			t.Fatalf("DeleteComment: %v", err)
		}
	})

	// A moderator editing someone else's comment must be recorded as the
	// moderator, not as the original author.
	t.Run("attributes edits to the editor", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		c, err := s.AddComment(ctx, tracker.NewComment{
			Item: item.ID, Author: agent("writer"), Body: "first",
		})
		if err != nil {
			t.Fatalf("AddComment: %v", err)
		}
		if _, err := s.EditComment(ctx, c.ID, c.Version, "edited", human("moderator")); err != nil {
			t.Fatalf("EditComment: %v", err)
		}
		page, err := s.Events(ctx, tracker.EventQuery{
			Item: &item.ID, Kinds: []tracker.EventKind{tracker.EventCommentEdited},
		})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(page.Rows) != 1 {
			t.Fatalf("edit events = %d, want 1", len(page.Rows))
		}
		if got := page.Rows[0].Actor; got.ID != "moderator" || got.Kind != tracker.ActorHuman {
			t.Errorf("Actor = %+v, want the editing moderator", got)
		}
	})

	// Comments written in the same instant must still page deterministically,
	// or a cursor can skip or repeat one.
	t.Run("orders totally", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		for _, body := range []string{"a", "b", "c", "d", "e"} {
			if _, err := s.AddComment(ctx, tracker.NewComment{Item: item.ID, Body: body}); err != nil {
				t.Fatalf("AddComment: %v", err)
			}
		}
		first, err := s.Comments(ctx, item.ID, tracker.PageRequest{})
		if err != nil {
			t.Fatalf("Comments: %v", err)
		}
		for range 5 {
			again, err := s.Comments(ctx, item.ID, tracker.PageRequest{})
			if err != nil {
				t.Fatalf("Comments: %v", err)
			}
			for i := range first.Rows {
				if again.Rows[i].ID != first.Rows[i].ID {
					t.Fatalf("comment order is not stable between reads")
				}
			}
		}
	})

	t.Run("refuses a comment on a missing item", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		_, err := s.AddComment(ctx, tracker.NewComment{Item: "ghost", Body: "x"})
		if !errors.Is(err, tracker.ErrNotFound) {
			t.Errorf("AddComment = %v, want ErrNotFound", err)
		}
	})
}

func runLinks(t *testing.T, newStore Factory) {
	t.Run("adds, lists and removes", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a := create(t, s, tracker.NewItem{})
		b := create(t, s, tracker.NewItem{})
		link := tracker.Link{From: a.ID, Kind: tracker.LinkBlocks, To: b.ID}

		if err := s.AddLink(ctx, link); err != nil {
			t.Fatalf("AddLink: %v", err)
		}
		// Links are visible from both ends.
		for _, end := range []tracker.ItemID{a.ID, b.ID} {
			links, err := s.Links(ctx, end)
			if err != nil {
				t.Fatalf("Links(%q): %v", end, err)
			}
			if len(links) != 1 {
				t.Errorf("Links(%q) = %d, want 1", end, len(links))
			}
		}
		if err := s.RemoveLink(ctx, link, human("vk")); err != nil {
			t.Fatalf("RemoveLink: %v", err)
		}
		if err := s.RemoveLink(ctx, link, human("vk")); !errors.Is(err, tracker.ErrNotFound) {
			t.Errorf("second RemoveLink = %v, want ErrNotFound", err)
		}
	})

	t.Run("adding twice is a no-op", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a := create(t, s, tracker.NewItem{})
		b := create(t, s, tracker.NewItem{})
		link := tracker.Link{From: a.ID, Kind: tracker.LinkRelates, To: b.ID}

		for range 2 {
			if err := s.AddLink(ctx, link); err != nil {
				t.Fatalf("AddLink: %v", err)
			}
		}
		links, err := s.Links(ctx, a.ID)
		if err != nil {
			t.Fatalf("Links: %v", err)
		}
		if len(links) != 1 {
			t.Errorf("Links = %d, want 1 after adding the same link twice", len(links))
		}
	})

	t.Run("refuses self-links and missing ends", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		a := create(t, s, tracker.NewItem{})
		if err := s.AddLink(ctx, tracker.Link{From: a.ID, Kind: tracker.LinkRelates, To: a.ID}); err == nil {
			t.Error("AddLink accepted a self-link")
		}
		err := s.AddLink(ctx, tracker.Link{From: a.ID, Kind: tracker.LinkRelates, To: "ghost"})
		if !errors.Is(err, tracker.ErrNotFound) {
			t.Errorf("AddLink to a missing item = %v, want ErrNotFound", err)
		}
	})
}
