# Spec: task tracker

Implementation spec for `services/api/internal/tracker`. Decisions live in
[ADR-0004](../adr/0004-task-tracker-domain-model.md) and
[ADR-0005](../adr/0005-storage-adapters-enforce-invariants.md); this is the
working document for what exists, what is next, and what is still open.

Status: **domain layer and an in-memory store implemented.** No durable
storage yet.

## Model

### Schema — configuration, edited at runtime

| Type         | Holds                                                                  |
| ------------ | ---------------------------------------------------------------------- |
| `Schema`     | every `ItemType`; the root of validation                               |
| `ItemType`   | its own fields, statuses, `Initial` status and transition graph        |
| `FieldDef`   | key, name, `FieldKind`, required, select options, default, type filter |
| `Status`     | key, display name, `StatusCategory`                                    |
| `Transition` | `From` → `To`, plus fields that must be set to make the move           |

`FieldKind` covers text, markdown, number, bool, date, duration, select,
multi-select, actor, item reference and URL.

`StatusCategory` is the closed set — `backlog`, `active`, `blocked`, `done`,
`canceled` — that gives user-defined statuses machine-readable meaning. `done`
and `canceled` are the resolved ones.

### Content

| Type      | Holds                                                                |
| --------- | -------------------------------------------------------------------- |
| `Item`    | type, title, body, status, `Parent`, `map[FieldKey]Value`, `Version` |
| `Comment` | author, body, optional one-level `ReplyTo`, `Version`                |
| `Link`    | `From` → `To` with a `LinkKind`; a graph over the tree, not the tree |
| `Event`   | append-only activity with a monotonic `Seq` and a list of `Change`   |

`NewItem` and `Patch` are the write inputs. `NewItem` has no status field —
creation enters `ItemType.Initial`. `Patch` uses nil pointers for "leave
alone", so clearing is distinguishable from not touching.

### Actors

`ActorRef{Kind, ID}` where kind is human, agent or system. Interchangeable
everywhere authorship appears.

## Hierarchy

A free tree. Any item may parent any other regardless of type, depth is
unbounded, and there are no fixed levels.

Two rules hold it together:

1. **Acyclic.** An item may not become its own descendant.
2. **Resolution gate.** An item may not enter `done` or `canceled` while any
   descendant is in an unresolved category. `SubtreeReader.UnresolvedDescendants`
   returning zero is the condition.

Reparenting moves the whole subtree.

## Ports

Small and single-purpose, so a consumer depends only on what it uses.

| Port                          | Methods                                       |
| ----------------------------- | --------------------------------------------- |
| `SchemaReader` `SchemaWriter` | read config; put/delete a type                |
| `ItemReader` `ItemFinder`     | by id; by `Query`                             |
| `ItemWriter`                  | create, update, delete — all compare-and-swap |
| `SubtreeReader`               | `UnresolvedDescendants`, `Ancestors`          |
| `CommentReader/Writer`        | list; add, edit, delete                       |
| `LinkReader/Writer`           | list; add, remove                             |
| `EventReader`                 | activity since a `Seq`                        |
| `Clock` `IDGenerator`         | ambient state, injected so tests stay boring  |

`Store` composes all of them. It is the adapter conformance target and the one
interface consumers must not depend on.

`Query` is set membership, field equality, one time range and a closed list of
sort keys. No expression language — a filesystem has to be able to answer it.

## Enforcement

Per ADR-0005, the adapter enforces. The invariants are listed in
`contract.go`: schema, workflow, hierarchy, resolution, concurrency.

The pure half is reusable and lives in the domain: `Validator` on `Schema`,
`ItemType`, `FieldDef`, `Status` and `Transition`, each validating itself and
everything downstream, plus `Schema.ValidateItem`, `ValidateNew`,
`ValidatePatch` and `ValidateTransition`. Validating the root validates the
tree.

The rules that must look at other items cannot come from a schema and belong
to the store: the tree stays acyclic, and the resolution gate. `memory`
implements them; a filesystem store will do the same in Go; a SQL store is
expected to push them into constraints and triggers and write no validation of
its own. All must produce the same sentinel errors.

### The resolution gate has three directions

An invariant that only holds when approached one way does not hold. All three
are enforced and tested:

1. An item may not **resolve** while any descendant is unresolved.
2. Unresolved work may not be **created or moved under** a resolved parent
   (`ErrResolvedParent`).
3. An item may not **reopen** beneath an already-resolved ancestor
   (`ErrResolvedParent`).

Canceling is gated exactly like completing — ADR-0004 refuses rather than
cascading. A canceled child does not hold its parent open, since `canceled` is
a resolved category.

## Implementations

**`memory`** — the fake ADR-0002 requires, enforcing every invariant itself.

**`trackertest`** — the conformance suite: 75 assertions across schema, items,
versioning, workflow, hierarchy, resolution, queries, comments, links, events
and isolation. Any adapter that has not passed it is not finished.

`Value` now has typed constructors and accessors, so a value that claims to be
a number and holds a string cannot be built. `Raw` remains for adapters
translating at their own boundary and is the only place the payload is loose.

## Decisions taken while implementing

- **Deleting an item with children is refused** (`ErrHasChildren`). Deleting
  would either orphan them or remove a subtree silently, and neither is a
  choice a delete call should make alone.
- **A stored schema may not be made to contradict its data.** `PutItemType`
  and `DeleteItemType` refuse a change that would invalidate existing items.
- **Comment threading is one level.** A reply cannot be replied to.
- **Adding the same link twice is a no-op**, not an error; the caller's intent
  is already satisfied.
- **`memory` cursors are offsets**, which is honest for a fake and wrong for
  anything durable — an offset shifts when rows are inserted. Cursors are
  opaque so each adapter can choose differently.

## Next

1. **Filesystem adapter** under `.oma/tracker/`, held to `trackertest`. This is
   the DB→filesystem swap ADR-0002 names as its reference case.
2. **HTTP surface**, mounted beside `/settings/`.
3. **Authentication**, before either is reachable from anywhere untrusted.

## Open questions

- **Cascade on cancel.** Canceling a parent with open children is currently
  refused by the resolution gate. Should it instead cancel the subtree? Refusing
  is safe but tedious for an agent abandoning a branch of work.
- **Comment deletion.** Hard or soft? Soft keeps `Event` history coherent.
- **Blocked as a category.** It overlaps with `LinkBlocks`. One of them may be
  redundant.
- **Schema migration.** `SchemaWriter` must reject changes that invalidate
  stored items. Whether it can also rewrite them is unanswered.
- **Reserved `@` prefix.** Convenient for `Change`, but it is a namespace rule
  every field key must respect forever.
