# Spec: task tracker

Implementation spec for `services/api/internal/tracker`. Decisions live in
[ADR-0004](../adr/0004-task-tracker-domain-model.md) and
[ADR-0005](../adr/0005-storage-adapters-enforce-invariants.md); this is the
working document for what exists, what is next, and what is still open.

Status: **types and ports only.** Nothing is implemented.

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

The filesystem adapter implements them in Go, leaning on `Validator`. The SQL
adapter pushes them into constraints and triggers and writes no validation of
its own. Both must produce the same sentinel errors.

## Next

1. **Conformance suite** (`trackertest`) — the executable specification. Per
   ADR-0005 this is not deferrable past the first adapter.
2. **`Validator` implementations** — `Schema`, `ItemType`, `FieldDef`, `Status`,
   `Transition`, each validating itself and everything downstream.
3. **`Value` constructors and accessors** — `Text("…")`, `v.Text() (string, bool)`.
   Until these exist the kind/payload invariant is convention only, and a wrong
   `Raw` is a runtime surprise. This is the largest open risk.
4. **In-memory adapter**, then filesystem, then SQL.

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
