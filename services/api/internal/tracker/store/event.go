package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// emit appends one event. Callers hold the write lock.
// A failed append is logged and swallowed. The change it describes has
// already been persisted and applied, so failing here would report a
// completed write as an error; the cost is a gap in the feed, which is
// exactly what readers already handle (ADR-0008).
func (s *Store) emit(
	ctx context.Context, id tracker.ItemID, kind tracker.EventKind,
	actor tracker.ActorRef, at time.Time, changes []tracker.Change,
) {
	s.seq++
	e := tracker.Event{
		ID:      tracker.EventID(s.ids.NewID()),
		Item:    id,
		Kind:    kind,
		Seq:     s.seq,
		Actor:   actor,
		At:      at,
		Changes: changes,
	}
	if err := s.disk.AppendEvent(ctx, e); err != nil {
		slog.Warn("cannot record a tracker event", "seq", e.Seq, "kind", kind, "err", err)
		return
	}
	s.events = append(s.events, e)
}

// emitChanges records an update as the events consumers actually wait on.
// Status and parent moves get their own kinds because reacting to "this was
// closed" should not mean sifting a generic update for a reserved key.
func (s *Store) emitChanges(ctx context.Context, before, after tracker.Item, actor tracker.ActorRef) {
	at := after.UpdatedAt
	if before.Status != after.Status {
		s.emit(ctx, after.ID, tracker.EventStatusChanged, actor, at, []tracker.Change{{
			Field: tracker.FieldStatus,
			From:  textOf(string(before.Status)),
			To:    textOf(string(after.Status)),
		}})
	}
	if !sameParent(before.Parent, after.Parent) {
		s.emit(ctx, after.ID, tracker.EventParentChanged, actor, at, []tracker.Change{{
			Field: tracker.FieldParent,
			From:  itemRefOf(before.Parent),
			To:    itemRefOf(after.Parent),
		}})
	}
	if changes := contentChanges(before, after); len(changes) > 0 {
		s.emit(ctx, after.ID, tracker.EventItemUpdated, actor, at, changes)
	}
}

// contentChanges diffs the title, body and custom fields.
func contentChanges(before, after tracker.Item) []tracker.Change {
	var out []tracker.Change
	if before.Title != after.Title {
		out = append(out, tracker.Change{
			Field: tracker.FieldTitle, From: textOf(before.Title), To: textOf(after.Title),
		})
	}
	if before.Body != after.Body {
		out = append(out, tracker.Change{
			Field: tracker.FieldBody, From: textOf(before.Body), To: textOf(after.Body),
		})
	}
	for key, now := range after.Fields {
		was, held := before.Fields[key]
		if held && was.Equal(now) {
			continue
		}
		change := tracker.Change{Field: key, To: valueOf(now)}
		if held {
			change.From = valueOf(was)
		}
		out = append(out, change)
	}
	for key, was := range before.Fields {
		if _, held := after.Fields[key]; !held {
			out = append(out, tracker.Change{Field: key, From: valueOf(was)})
		}
	}
	return out
}

// Events implements [tracker.EventReader]. Ordering is by sequence, which is
// how a reader resumes without re-handling what it has already seen.
func (s *Store) Events(ctx context.Context, q tracker.EventQuery) (tracker.EventPage, error) {
	if err := ctx.Err(); err != nil {
		return tracker.EventPage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched []tracker.Event
	for _, e := range s.events {
		if e.Seq <= q.Since {
			continue
		}
		if q.Item != nil && e.Item != *q.Item {
			continue
		}
		if len(q.Kinds) > 0 && !containsKind(q.Kinds, e.Kind) {
			continue
		}
		matched = append(matched, cloneEvent(e))
	}
	return paginate(matched, q.Page)
}

// cloneEvent copies the changes an event carries. Without it a caller could
// rewrite recorded history through the pointers it was handed.
func cloneEvent(e tracker.Event) tracker.Event {
	out := e
	out.Changes = make([]tracker.Change, 0, len(e.Changes))
	for _, c := range e.Changes {
		if c.From != nil {
			from := *c.From
			c.From = &from
		}
		if c.To != nil {
			to := *c.To
			c.To = &to
		}
		out.Changes = append(out.Changes, c)
	}
	if len(out.Changes) == 0 {
		out.Changes = nil
	}
	return out
}

func containsKind(kinds []tracker.EventKind, k tracker.EventKind) bool {
	for _, want := range kinds {
		if want == k {
			return true
		}
	}
	return false
}

func sameParent(a, b *tracker.ItemID) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

func textOf(s string) *tracker.Value {
	v := tracker.Text(s)
	return &v
}

func valueOf(v tracker.Value) *tracker.Value { return &v }

func itemRefOf(id *tracker.ItemID) *tracker.Value {
	if id == nil {
		return nil
	}
	v := tracker.ItemRef(*id)
	return &v
}
