# ADR-0016: Query widens for fields and sorting, and stops before comparators

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/api/internal/trackerhttp`
- Relates to: ADR-0001, ADR-0002, ADR-0004

## Context

Most of what a tracker needs is already storable, because ADR-0004 made fields
runtime data: assignee is a `KindActor` field, priority a select, estimates a
duration, due dates a date. None of it is findable.

`Query.Fields` exists in the domain and is deliberately absent from HTTP,
because a field match needs a typed value and a string cannot be turned into
one without knowing the field's kind. `SortKey` is a closed list of created,
updated and title, so a priority field can be stored and never ordered by.

A tracker where an agent cannot ask "what is assigned to me and not done" is
one agents cannot work from. The temptation is to answer that with a query
language, and ADR-0002 is what stops it: a filesystem store must be able to
answer anything the port offers.

## Decision

**Field matches over HTTP.** The edge looks up the field's kind in the schema
and builds the right `Value`. That the HTTP layer now needs the schema is a
real cost, accepted because this is the single highest-value gap for agents and
the domain already supports the match.

**Sorting widens to orderable field kinds** — number, date and duration. That
is what makes priority and due date usable at all, and it stays a bounded set
rather than an expression.

**A substring filter over title and body**, ANDed with everything else, with no
ranking. Any backend can answer it by reading what it already reads.

**No range comparators.** "Due before Friday" is refused for now. This is the
piece most likely to grow into a query language — one comparator on one kind is
never where it stops — and nothing yet needs it. A caller wanting overdue items
can fetch the project's items and filter.

**No ranked or fuzzy search, ever.** It cannot be honoured by a filesystem
adapter without an index, which ADR-0002 forbids as a difference between
adapters. Agents query structurally, not by recall.

## Consequences

**Every widening is a promise every future adapter must keep.** A SQL store
would find all of this trivial. The filesystem store answers a substring filter
by reading every item in the project, which is honest and bounded by how large
one project gets — and if that stops being true, the answer is an index behind
the same port, not a narrower port.

**The HTTP edge now depends on the schema**, which it did not before. That is a
new coupling in a layer that had none, and the reason it is acceptable is that
`scopes` already hands out both from one place.

**Refusing comparators will be felt.** "Due this week" is an obvious thing to
want, and the answer for now is that the client filters. If that becomes the
common case rather than the rare one, this decision is worth revisiting with
evidence rather than anticipation.
