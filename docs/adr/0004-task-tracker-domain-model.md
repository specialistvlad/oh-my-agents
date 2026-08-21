# ADR-0004: The task tracker is one generic item type with a runtime schema

- Status: Accepted; the schema-key part is superseded by ADR-0009
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`
- Relates to: ADR-0002, ADR-0003

## Context

The tracker is the substrate the whole system runs on: agents pick work up
from it, report against it, and decompose it into more work. It has to carry
epics, features, user stories, requirements, releases and bugs, and the list
of what it carries is not knowable now.

Every one of those classes wants its own fields and its own columns. A release
does not have reproduction steps; a bug does not have a cutover date. Teams
rename columns. This is the ClickUp/Jira problem, and it has a known shape.

## Decision

**One generic `Item`. Schema is data.** An item holds a `TypeKey`, a
`StatusKey` and `map[FieldKey]Value`; an `ItemType` declares which statuses and
fields are legal for it. Adding "incident" with six fields is a write. The cost
is accepted: no compiler check that a field exists, paid back by never
redeploying to add a type.

**Statuses are user-defined; categories are not.** Every `Status` carries a
`StatusCategory` from a closed set — backlog, active, blocked, done, canceled.
Logic asks the category. Nothing in the system may branch on a status key or
name, because those belong to whoever configured the tracker, and a rename must
not break orchestration.

**Hierarchy is a free tree of unbounded depth.** Any item may parent any other,
regardless of type. The fixed epic → story → subtask ladder is not modeled: an
agent decomposing work does not know in advance how deep it will need to go, and
a ladder would force it to lie about what a node is.

**A parent cannot resolve before its descendants.** Entering `done` or
`canceled` is refused while any descendant sits in an unresolved category.
Closing a parent is an assertion that everything beneath it is settled, and the
tracker makes that assertion true rather than trusting it. Canceled counts as
resolved — an abandoned child does not hold its parent open forever.

**Transitions are always enforced.** Every type declares its allowed moves;
anything else is rejected. Creation is the one exception and enters
`ItemType.Initial` without consulting the graph, so creation cannot be used to
sidestep a workflow.

**Every write is compare-and-swap** on `Version`. Concurrent agents on one item
is the normal case, not the edge case, and last-write-wins loses work silently.

**Agents are ordinary actors.** `ActorRef` spans human, agent and system, and
authorship accepts any of them everywhere. An agent narrating progress posts a
comment, on the same path a person does.

## Consequences

Nothing about an item can be understood without its type. Reads that render or
validate an item need the schema in hand, so `SchemaReader` is a dependency of
almost everything.

Queries stay narrow — set membership, field equality, one time range, a closed
list of sort keys — because ADR-0002 makes a filesystem implementation the
yardstick. Anything wanting a real query language will have to widen the port
deliberately, having first answered how files on disk honor it.

`Value` is dynamically typed at the edges. It currently exposes `Kind` and
`Raw any` with the invariant held by convention; typed constructors and
accessors land with the validator. Until then a wrong `Raw` is a runtime
surprise, which is the main open risk in this design.

Reserved `@` field keys (`@title`, `@status`, `@parent`, `@body`) let a
`Change` describe built-in edits in the same shape as custom-field edits, at
the cost of a namespace rule field keys must respect.
