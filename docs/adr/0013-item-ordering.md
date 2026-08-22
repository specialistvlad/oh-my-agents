# ADR-0013: Items carry one global rank, and reordering is not an edit

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/web`
- Relates to: ADR-0001, ADR-0002, ADR-0004, ADR-0008

## Context

A board's defining interaction is dragging a card to a _position_. Nothing in
the domain can express one: `Item` has no rank, `Query.Sort` is a closed list
of created, updated and title, and `Patch` has no field for reordering.

An ordering bolted on at the web layer would be per-browser, invisible to
agents, and lost on reload. So it belongs in the domain, and the question is
what shape it takes.

## Decision

**One rank per item, global to the project.** Not per status, and not per
status and parent. A column's order falls out of filtering the global order,
because the relative order of any subset is preserved by the order it is drawn
from. Per-status ranking would mean every status change also reassigns a rank —
a second thing to get right for no gain a person can see.

**Rank is a sparse lexicographic key**, so inserting between two neighbours
writes one item. The filesystem store is the yardstick (ADR-0002): one file per
item means a dense integer position rewrites every sibling on every drag, and a
sparse key rewrites exactly one.

**Reordering is its own operation and does not go through `Patch`.** `Patch` has
no rank field. A drag is not an edit in any sense a person would recognise, and
ADR-0001 prefers a small single-purpose unit over a flag on a general one.

**Reordering does not bump `Version`.** This is the part that needed the most
care, and the code already makes it safe: `UpdateItem` re-reads the item inside
the lock and applies the patch on top of what it finds. So an edit saved after
a drag cannot revert the drag — the patch never carries a rank to revert it
with. A drag and an edit therefore do not conflict, which is the right answer,
because they do not overlap.

**Two clients dragging one card is last-write-wins, silently.** No conflict is
reported. A drag states a position rather than a transformation of one, so
there is nothing to merge and nothing the loser needs to be told.

## Consequences

**Rank is the first thing in this model with no natural default.** An item has a
type, a status and a parent because something supplied them. A new item's rank
has to be invented, so creation now answers "where does this go" — at the end,
which is the only answer that does not surprise someone.

**`Version` keeps one meaning.** It is the version of the item's _content_.
Dragging cards around a board all afternoon produces no version churn and
conflicts with nothing, which is what makes a board usable by several people
at once.

**Ordering is an invariant every adapter must enforce**, so `contract.go` gains
it and `trackertest` gains assertions for it (ADR-0005). The memory and
filesystem stores share their enforcement, so what the suite really proves here
is that rank survives a restart.

**A sparse key can run out of room.** Repeatedly inserting between the same two
neighbours lengthens the key until it needs rebalancing. That is a real limit,
deferred deliberately: it takes thousands of insertions at one spot, and the
fix — renumber a project's items once — is available whenever it matters.
