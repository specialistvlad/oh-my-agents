# ADR-0017: A column is a status, a board is one type, and a view is local

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/web`
- Relates to: ADR-0004, ADR-0008, ADR-0009, ADR-0011, ADR-0012, ADR-0013

## Context

The app shows a flat list. A board needs three things the list did not:
ordering, which ADR-0013 settles; a definition of what a column is; and an
answer to whether a saved view is something the system stores.

## Decision

**A column is a status.** Not an arbitrary groupable field. ADR-0004 already
gives every status a category and a declared set of transitions, so a board
made of statuses _is_ a view of the workflow: the columns are the states and
the arrows between them are already written down. Grouping by an arbitrary
field would be a more general board that says nothing about how work moves, and
it would need answers for what an empty group is, what order groups appear in,
and what a drag between them means when it is not a transition.

**A board shows one type.** Two types with different status sets have no
reliable notion of "the same column" — matching by name is a coincidence, and
category is too coarse to be a column. One type keeps a column unambiguous.

**An illegal drop is never a drop target.** ADR-0004 enforces transitions
always, so a board could refuse the drop and snap the card back. Not offering
it is better and costs more: legality is per _item_, because the card's current
status decides what it can reach, so the board computes reachable statuses per
card rather than per type.

**A view is local, like the layout.** Filters, grouping and column order go to
the browser (ADR-0011). This is the decision ADR-0012 forces: with no
authentication there is nobody to own a personal view, so a stored view would
be a _shared_ one — and then anyone reaching the server can change what
everybody sees. Local views need no API, no domain concept, and no versioning.

## Consequences

**Two people looking at "the board" may be looking at different boards**,
because each has their own filters. That is the cost of local views, and it is
the same trade ADR-0011 already made for column widths. It stops being right
the moment a team wants a shared board they all recognise — at which point a
stored view is a new decision, not a gap.

**Choosing status-as-column ties the board to the workflow**, which is what
makes it legible and also what limits it. A board grouped by assignee is now a
different feature rather than a configuration of this one.

**Computing reachable statuses per card is more work per render** than checking
once per type. It is what keeps the board from offering moves that always fail,
which is the kind of dead control that teaches people not to trust an
interface.

**Nothing here widens `Query`.** ADR-0016 widens it for agents, separately and
for its own reasons. A local view filters what it already fetched.
