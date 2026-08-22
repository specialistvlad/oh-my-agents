# ADR-0017: Boards, grouping, and where a view lives

- Status: **Proposed** — the questions below need answering before this is settled
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/web`
- Relates to: ADR-0004, ADR-0008, ADR-0009, ADR-0011, ADR-0012, ADR-0013

## Context

The app shows a flat list. A board is the shape most of this work exists to
produce, and it needs three things settled that the list did not.

Ordering is one of them and has its own decision (ADR-0013). The other two are
what a column _is_, and whether a saved view is a thing the system stores.

## What has to be decided

**What a column is.** The natural answer is a status, because ADR-0004 already
gives every status a category and a declared set of transitions between them —
a board would then be a view of a type's workflow. The other answer is any
field: group by assignee, by a select field, by priority. That is a more
general board and a much larger surface, because grouping by an arbitrary field
means deciding what an empty group is, what order groups appear in, and what a
drag between them _means_ when it is not a transition.

**What a board does with an illegal drop.** ADR-0004 enforces transitions
always. So either the board refuses the drop and snaps the card back, or it
never renders an illegal column as a drop target. The second is better and
costs more: legality is per item, not per type, because one board can hold
items of several types with different workflows.

**Whether a board spans types.** If it does, "the same column" across two types
with different status sets needs a definition — matching by name, by category,
or not at all. Category is the only one the domain actually guarantees.

**Where a view lives.** This is the question ADR-0012 makes strange. A saved
view is normally _mine_ — my filter, my grouping, my column order. There is no
authentication and no user, so there is nobody to own one. That leaves two
honest answers: a view is a **shared project object**, versioned and announced
like everything else, and everyone sees the same ones; or a view is **local UI
state** like column widths (ADR-0011), and the system stores nothing.

The second needs no API at all. The first makes views the first collaborative
object outside the tracker's own model.

## Questions to settle

1. Is a column a status, or any groupable field?
2. Is a board scoped to one type, or can it span several — and if it spans, what
   makes two statuses "the same column"?
3. Are saved views shared project objects with full event and version
   machinery, or local like layout?
4. If views are shared, does `Query` need widening to express what they filter
   on — which is ADR-0016's question arriving from a second direction?

## Consequences either way

Choosing status-as-column keeps a board a view of the workflow, which is what
makes the transition rules legible: the columns _are_ the states, and the
arrows between them are already declared. Choosing any-field makes the board
generic and detaches it from the workflow it displays.

Choosing local views keeps the domain small and means two people looking at
"the board" may be looking at different boards. Choosing shared views means a
person can change what everybody sees, which with no authentication (ADR-0012)
is a thing anyone reaching the server can do.
