# ADR-0014: Editing an item under compare-and-swap

- Status: **Proposed** — the questions below need answering before this is settled
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`, `services/web`
- Relates to: ADR-0004, ADR-0008, ADR-0011, ADR-0012

## Context

Nothing in the app can edit an item. A status can be moved; a title, a body and
every custom field are read-only, and the frontend `Item` type does not even
carry `fields`.

Making them editable runs straight into the concurrency model. Every write is
compare-and-swap on `Version` (ADR-0004), there is no operational transform or
CRDT anywhere (ADR-0008), and `Version` is **one counter per item**. So two
people editing different fields of the same item collide, though nothing they
did overlaps.

A markdown body makes this sharper than a title does. Losing a word is
annoying; losing three paragraphs someone was mid-way through writing is the
kind of thing that stops people trusting an application.

## What has to be decided

**Whether `Version` stays per item.** Per-field versioning — or a patch that
carries only what actually changed and is checked field by field — would let
two people edit different parts of one item. It also gives the store two kinds
of version to keep consistent, and gives every adapter more to enforce.

**What a rejected save does to the draft.** Silently re-reading loses
keystrokes. Blocking with "someone else saved" keeps the draft but stops the
person until they resolve it. A field-level merge is the most forgiving and the
most machinery.

**Where the browser's identity comes from.** ADR-0012 makes actors
self-declared, and the browser currently declares nobody — every UI write is
attributed to an empty actor. Something must supply one: a name typed once and
remembered per device, the way layout is (ADR-0011). The UI must not imply this
is verified, because it is not and cannot be.

**How typed fields are edited.** There are eleven kinds. Which invalid states
are caught in the browser (URL shape, number parsing, select membership) and
which must round-trip (item-type narrowing on `KindItem`, required fields).
`KindActor` and `KindItem` need pickers, which need data — and ADR-0011 says
nothing in a component fetches.

## Questions to settle

1. Does `Version` stay one counter per item, accepting that two people editing
   different fields conflict?
2. On a conflict while editing a body, is the draft preserved, and where?
3. Does the person choose an identity, or does the browser invent one they
   never see? An invented one makes the feed useless; a chosen one is a prompt
   nobody asked for on first load.
4. Are `KindActor` and `KindItem` free text, or pickers — and if pickers, which
   layer fetches their options?

## Consequences either way

The activity feed becomes meaningful for the first time, because writes will
carry an actor. It also becomes the first place one person's self-declared name
is shown to another, which ADR-0012 says is a claim and not evidence. The UI
has to not dress it up as more than that.
