# ADR-0014: Editing keeps one version per item, and never loses a draft

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/web`
- Relates to: ADR-0001, ADR-0004, ADR-0008, ADR-0011, ADR-0012

## Context

Nothing in the app can edit an item beyond moving it between statuses. Making
title, body and custom fields editable runs into the concurrency model: every
write is compare-and-swap on `Version` (ADR-0004), there is no operational
transform or CRDT (ADR-0008), and `Version` is one counter per item.

So two people editing _different fields_ of one item collide, though nothing
they did overlaps. The instinct is to fix that with per-field versioning.

## Decision

**`Version` stays one counter per item.** Per-field versioning is a second
concurrency story for every adapter to enforce and every edge to explain, in
exchange for a case that is uncommon in a tracker.

**The stricter part is the check, not the data.** `UpdateItem` re-reads the item
inside the lock and applies the patch on top, so a patch that sets only `title`
cannot clobber a `body` written a moment earlier. The version check refuses
writes that would in fact have been safe.

**So a conflict is resolved by the client, not the domain.** On
`ErrVersionConflict` the client refetches and, if the fields it is writing are
unchanged, re-applies its patch silently. If they did change, it stops and
shows the other version. The common case resolves itself; the real case asks a
person. This is a client policy, and nothing in the domain grows to support it.

**A draft is never lost.** Whatever the conflict, what someone typed stays in
the tab until they discard it. Losing a word is annoying; losing three
paragraphs someone was mid-way through is what stops people trusting an
application, and no merge strategy is worth that risk.

**Identity is a name the browser claims**, stored per device like the layout
(ADR-0011), defaulting to something visibly generic rather than empty. It is a
claim and not evidence (ADR-0012), and the UI says so rather than presenting it
as an account.

**`KindActor` and `KindItem` are edited as free text for now.** Pickers need
data, and ADR-0011 says nothing in a component fetches — so a picker means a
hook, a fetch and a cache for a field nobody has yet used. It waits for a
caller.

## Consequences

**Two people editing different fields of one item will sometimes see a
conflict that resolves itself**, because the client retries. They will
occasionally see one that does not, and be shown the other version. That is the
accepted cost of one version counter.

**A retry that re-applies silently is a write nobody asked to repeat.** It is
safe because the patch is the same and the fields it touches are unchanged, but
it means an edit can land later than the person who made it believes. The event
feed records when it actually landed.

**Client-side validation is a convenience, never the guard.** The store already
refuses a value whose kind contradicts its field (ADR-0005), and the browser
checking first only saves a round trip.
