# Spec: what the tracker needs to be useful

The tracker has a domain model, two edges, durable storage and a workspace to
show it in. What it does not have is a product: no board, no way to configure
anything, no way to edit an item beyond moving it between statuses.

This is the survey of that gap — what is missing, what only looks missing, and
what is deliberately refused. The decisions it led to are
[ADR-0013](../adr/0013-item-ordering.md) through
[ADR-0017](../adr/0017-boards-and-views.md).

## Three defects, since fixed

These were gaps in what was already built rather than decisions, and everything
below depended on them. Recorded because what they turned out to be is worth
remembering.

**Minting was written twice.** `tracker.Mint` was called from nowhere, which
looked like dead code waiting for a schema editor. It was really a second copy
of `projects.MintID` — same stem reduction, same length bound, same regexp, in
two packages. Both now delegate to `internal/ids`, so there is one grammar for
identifiers rather than two that would drift.

**The browser declared nobody.** Every write from the UI carried an empty
actor, so the activity feed recorded nothing about who did what. The browser
now claims a name, stored per device like the layout, and every tracker write
carries it. It is a claim and not evidence (ADR-0012), and the control says so
rather than dressing it up as an account.

**The frontend could not see custom fields.** `Item` had no `fields` at all, so
values round-tripped through the API and the filesystem and were invisible to
the app — ADR-0004's runtime schema was unreachable from the UI. The inspector
now shows every field a type declares, including the ones nobody has filled in,
because a field that only appears once it has a value is a field nobody
discovers.

## What real trackers have

Most of it is already expressible. The domain deliberately has one generic
`Item` with runtime-defined fields (ADR-0004), and that covers more than it
appears to.

| Capability                | Verdict                                    | Why                                                                                                                   |
| ------------------------- | ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------- |
| Assignee, reviewer, owner | expressible                                | `KindActor` field                                                                                                     |
| Priority                  | expressible; **sorting is the gap**        | `KindSelect` or `KindNumber` stores it, but `SortKey` is a closed list of created/updated/title                       |
| Due dates                 | expressible; **range queries are the gap** | `KindDate` stores it; `FieldMatch` is equality-only, so "due this week" is unanswerable                               |
| Labels and tags           | expressible                                | `KindMultiSelect`. There is no global tag namespace, which is arguably a feature                                      |
| Estimates                 | expressible                                | `KindDuration`                                                                                                        |
| Claiming work (agents)    | **already solved**                         | Compare-and-swap on `Version` is a race-free claim primitive. No new verb needed                                      |
| Activity digests          | expressible                                | `EventReader` since a `Seq` is the raw material; summarising is a consumer's job                                      |
| Archiving                 | mostly expressible                         | `Query.Categories` already excludes done and canceled                                                                 |
| Free-text search          | **narrow subset only**                     | A substring match over title and body is answerable by any backend. Ranking and fuzzy matching are not — see ADR-0016 |
| Ordering within a column  | **missing entirely**                       | No rank exists anywhere in the model. See ADR-0013                                                                    |
| Editing anything          | **missing entirely**                       | Only status moves are possible today. See ADR-0014                                                                    |
| Configuring anything      | **missing entirely**                       | The schema is seeded and read-only. See ADR-0015                                                                      |

## Deliberately refused

Recorded so they are not proposed again. Each is refused on ADR-0001 grounds —
do not build what is not needed — or because it contradicts a decision already
taken.

**Bulk operations.** Sugar over a loop. No cross-item atomicity exists or is
promised: `Version` is per item, so a bulk endpoint would either lie about
atomicity or need a transaction concept the storage layer does not have.

**Watchers and notifications.** A permission model in disguise. The bus
broadcasts everything to everyone by design (ADR-0008), and ADR-0012 has no
identity to scope a subscription to. This would be the system's first
per-actor filter.

**Attachments.** No blob store exists anywhere, and an unauthenticated server
(ADR-0012) hosting arbitrary binaries is new attack surface for a need
`KindURL` half covers.

**Recurring items.** No scheduler or clock trigger exists in the system, and an
agent creating an item on a cadence reproduces it exactly.

**Item templates.** `FieldDef.Default` covers field-level defaulting. Anything
beyond that is a client convention, not stored state.

**Ranked or fuzzy search.** Cannot be honoured uniformly by a filesystem
adapter without an index, which ADR-0002 makes the yardstick. Agents query
structurally — by status, category, assignee — not by vague recall.

## One open contention

**Human-readable item numbers** (`PROJ-123`). People want them badly: in commit
messages, in PR titles, in conversation. They contradict ADR-0009, where IDs are
opaque and nothing may parse or infer from one.

A sequential display number is a _new concept_ rather than a change to IDs, so
it can be added without reversing ADR-0009 — but it needs deciding rather than
defaulting either way. It is attached to none of ADR-0013 to ADR-0017 and
remains open.

## Buildable today, with no decisions

- A read-only status-grouped board: columns from `ItemType.Statuses`, cards
  ordered by an existing sort key, click-to-move gated by the declared
  transitions the UI already checks. A kanban shape without kanban's ordering.
- Comment display. `GET …/items/{item}/comments` exists and has no UI.
- Read-only rendering of custom field values, once the frontend `Item` type
  carries them.
- A read-only hierarchy view, using `Query.Parent` and `Query.Subtree`.
