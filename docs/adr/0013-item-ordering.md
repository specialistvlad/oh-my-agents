# ADR-0013: Items carry an explicit order

- Status: **Proposed** — the questions below need answering before this is settled
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/web`
- Relates to: ADR-0002, ADR-0004, ADR-0008

## Context

A board's defining interaction is dragging a card to a _position_ within a
column. Nothing in the domain can express one.

`Item` has no rank or position field. `Query.Sort` is a closed list —
`created_at`, `updated_at`, `title` — with no manual-order key and no way to
add one from outside the package. `Patch` has no field for reordering.

So this is a domain-model gap, not a UI gap, and no board can be built over it.
It is also the gap most likely to be papered over badly: an ordering bolted on
at the web layer would be per-browser, invisible to agents, and lost on reload.

## What has to be decided

**What rank is.** A sparse fractional or lexicographic key (insert between two
neighbours without touching them) or a dense integer position (insert means
renumbering the siblings). The filesystem store is the yardstick (ADR-0002):
one file per item means a dense scheme rewrites every sibling on every drag,
while a sparse one rewrites exactly one file.

**What it is scoped to.** Global per project, per status, or per status and
parent together. Per-status ordering means a status change must also assign a
new rank, because the card is arriving at a position in a different column.

**Whether reordering bumps `Item.Version`.** This is the sharp one. Every write
is compare-and-swap on `Version` (ADR-0004). If a drag bumps it, then dragging
a card conflicts with somebody editing that card's description, though the two
do not overlap semantically. If it does not, position needs its own versioning
and the store has two concurrency stories instead of one.

**Whether reordering is its own operation.** `UpdateItem` with a `Patch` is the
existing write path, but ADR-0001 prefers small single-purpose units, and a
drag is not an edit in any sense a person would recognise.

## Questions to settle

1. Is rank per-status, or global across the project? Per-status is what a board
   wants; global is one number and no reassignment on a status change.
2. Does a drag bump `Item.Version`? Answering yes keeps one concurrency story
   and makes drag storms conflict with content edits. Answering no needs a
   second version and a reason it cannot drift from the first.
3. When two clients drag the same card to different places at once, what does
   the loser see — its card snapping back, or landing where the winner put it?
4. Is reordering a separate port method, or a field on `Patch`?

## Consequences either way

Whatever is chosen, `contract.go` gains an invariant, and `trackertest` gains
assertions — ADR-0005 requires every adapter to enforce ordering identically,
and the suite is what says they do.

Ordering is also the first thing in this model with no natural default. An item
has a type, a status and a parent because something set them; a rank has to be
invented at creation, and "where does a new item go" becomes a question the
domain has to answer rather than dodge.
