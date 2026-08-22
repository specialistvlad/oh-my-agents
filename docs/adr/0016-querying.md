# ADR-0016: How far the query surface widens

- Status: **Proposed** — the questions below need answering before this is settled
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/api/internal/trackerhttp`
- Relates to: ADR-0001, ADR-0002, ADR-0004

## Context

Most of what a tracker needs — assignee, priority, labels, estimates, due dates
— is already storable, because ADR-0004 made fields runtime data. None of it is
findable.

- `Query.Fields []FieldMatch` exists in the domain and is **deliberately absent
  from HTTP**, because a field match needs a typed value and there is no honest
  way to read one from a string without knowing the field's kind.
- `FieldMatch` is equality-only, so "due before Friday" cannot be asked even
  in Go.
- `SortKey` is a closed list of created, updated and title, so a priority field
  can be stored and never ordered by.

So a tracker where an agent cannot ask "what is assigned to me and not done" is
a tracker agents cannot work from. That is the gap worth closing, and the
temptation is to close it with a query language.

ADR-0002 is the constraint: a filesystem store must be able to answer anything
the port offers. That rules out ranked search and indexes, and it is what keeps
this from becoming a database.

## What has to be decided

**Typed field matches over HTTP.** The domain already supports them. What is
missing is schema-aware parsing at the edge: given `?field.priority=high`, the
handler must look up the field's kind to build the right `Value`. That is a
real cost — the HTTP layer starts needing the schema — and it is the single
highest-value gap for agents.

**Ordering by a field.** Widening `SortKey` from a fixed list to a bounded set
of orderable kinds — number, date, duration — makes priority and due date
usable without an expression language.

**Range comparators.** "Due before X" needs more than equality on at least date
and number. Adding one comparator to some kinds is a small change that is hard
to stop growing.

**A substring filter.** Title and body, ANDed with everything else, no ranking.
Any backend can answer it by reading what it already reads. Ranking and fuzzy
matching cannot, and are refused (see the product spec).

## Questions to settle

1. Do field matches over HTTP happen, given the edge must then know the schema
   to parse a value?
2. Does `SortKey` widen to orderable field kinds, or stay closed?
3. Do range comparators arrive now, or wait for something that needs them? They
   are the thing most likely to grow into a query language.
4. Is a substring filter on title and body worth it, or is fetch-and-filter
   good enough at this scale?

## Consequences either way

Every widening is a promise every future adapter must keep. A SQL store would
find range comparators trivial; the filesystem store answers them by reading
every item, which is honest but bounded by how large a project gets.

Refusing all of it is also a real option, and cheaper than it sounds: a client
can fetch a project's items and filter locally, and at the scale one project
reaches, that may be indistinguishable.
