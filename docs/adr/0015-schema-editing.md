# ADR-0015: Schema edits are granular, server-minted, and never silently lossy

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/api/internal/trackerhttp`, `services/web`
- Relates to: ADR-0001, ADR-0004, ADR-0005, ADR-0009

## Context

ADR-0004's premise is that types, fields and statuses are data, so a new class
of work costs a write rather than a deploy. Nothing can perform that write. A
project is seeded with one Task type and the only API is `PUT`/`DELETE` of a
whole type — an API designed for a machine writing a schema atomically, not for
a person adding one field.

Whole-type replacement also means two people editing different parts of one
type overwrite each other wholesale, and it makes the client responsible for
reconstructing a valid type from a form.

## Decision

**Granular operations alongside whole-type `PUT`.** Add and edit a field, add a
status, add and remove a transition, each its own operation the server
validates and applies. `PUT` stays for import and machine use, where writing a
whole type at once is exactly right.

**The server mints identifiers, on save.** An add-field call takes a name and
returns the minted id. The client never invents one, which is what ADR-0009
wanted and what `validID`'s shape check cannot enforce on its own. Minting on
save rather than as someone types means an abandoned draft leaves no orphan id
behind.

**Deleting a status in use requires a replacement.** The call names where its
items go. Refusing forever is a control that can never succeed; reassigning
automatically changes data without being asked. Naming a replacement is the
only option that is both explicit and always has a way forward.

**Removing a field that holds values requires acknowledgement.** The first call
fails and says how many items hold one; repeating it with an explicit discard
succeeds. No separate dry-run endpoint — the refusal _is_ the preview, which is
one mechanism instead of two.

**Adding `required` binds future writes only.** Existing items are not
retroactively invalid, and the requirement bites the next time one is written.
The alternative — refusing the schema change until every item is backfilled —
blocks a legitimate change on data nobody may have.

**A type with no transitions is a validation error.** `schema.go` already calls
it "a configuration error rather than a special case"; this makes that an
enforced rule rather than a comment. A type you can create but never move is
never what anyone meant.

**Templates stay seed data.** Richer starting schemas are a change to
`scopes.seed`, not a concept the domain needs to know about (ADR-0001).

## Consequences

**An item can be left unable to save until someone fills in a field.** That
follows directly from `required` binding future writes: an item created before
the rule cannot be edited without satisfying it. The UI has to say which field
and why, or that item looks broken.

**The schema becomes the first thing a person can break for everybody in a
project.** It is project-scoped (ADR-0009) so it cannot escape one, but within
it a removed status arrives at every client as an event, mid-interaction.

**Every granular operation is another invariant every adapter enforces**
(ADR-0005), so each arrives with assertions in `trackertest` rather than only
where it is implemented.

**Two people editing one type can still collide**, because the type is the unit
of storage even when the operation is granular. Granularity buys a smaller
window and a clearer intent, not immunity.
