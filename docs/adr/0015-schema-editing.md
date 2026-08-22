# ADR-0015: Configuring a tracker from the UI

- Status: **Proposed** — the questions below need answering before this is settled
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/api/internal/trackerhttp`, `services/web`
- Relates to: ADR-0001, ADR-0004, ADR-0005, ADR-0009

## Context

ADR-0004's premise is that types, fields and statuses are **data**, so adding a
class of work costs a write rather than a deploy. Nothing can perform that
write. A project is seeded with one hardcoded Task type, and the only API is
`PUT`/`DELETE` of a whole type.

That API was designed for a machine writing a schema atomically — seeding, or
an import. It is the wrong shape for a person adding one field:

- The client must reconstruct an entire `ItemType` and resubmit it, so two
  people editing different parts of one type overwrite each other wholesale.
- Adding a status means inventing its ID. `tracker.Mint` exists for exactly
  this and **is called from nowhere**; `validID` only checks an ID looks minted,
  not that it was.
- `PutItemType` refuses any change that would invalidate stored items. There is
  no migration path — only rejection — so a UI can offer a control that is
  permanently refused with no way forward.

There is also a latent trap: **adding `required` to a field would make every
existing item fail `ValidateItem`** on next read. Harmless while nothing edits
schemas; a data-loss-shaped bug the moment something does.

## What has to be decided

**Granular operations, or whole-type replacement.** Add-field, rename-option,
add-transition as their own operations that the server validates and applies —
against keeping `PUT` and having the client rebuild the type. `PUT` stays
either way for import and machine use.

**Who mints IDs.** If the server does, add-field takes a name and returns an
ID, and `Mint` finally has a caller. If the client does, `validID`'s shape
check becomes the only guard against hand-invented IDs, which ADR-0009 exists
to prevent.

**Refuse, or migrate.** Deleting a status with items in it, removing a field
holding data, dropping a select option in use, adding a required field. Each
needs an answer, and "refuse forever" is a legitimate one — but it has to be
chosen rather than inherited from the current code.

**What a legal workflow is.** `schema.go` already notes in a comment that a
type with no transitions "can be created but never moved, which is a
configuration error rather than a special case." That needs to become an
enforced rule or an accepted state. Unreachable statuses and a dangling
`RequiredFields` reference need the same treatment.

## Questions to settle

1. Deleting a status with items in it: refuse, require a replacement, or
   reassign automatically?
2. Removing a field that holds values: silent loss, or a confirmation that
   needs a dry-run endpoint to say "12 items will lose this"?
3. Adding `required` to a type with existing items: are they now invalid, or
   does the rule apply only to new writes?
4. When are IDs minted — as a person types a draft field, or on save? Minting
   early leaves orphan IDs behind every abandoned edit.
5. Is "no transitions declared" a hard validation error?
6. Should a project be able to start from something richer than one Task type,
   and is that a decision or just seed data?

## Consequences either way

The schema becomes the first thing in the system a person can break for
everybody. It is project-scoped (ADR-0009), so a bad edit does not escape one
project — but within it, a removed status is removed for every client watching,
arriving as an event mid-interaction.

Whatever migration answer is chosen, `trackertest` gains assertions for it,
because ADR-0005 makes every adapter enforce it identically.
