# ADR-0009: Everything is scoped to a project and addressed by a minted ID

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api`, `services/web`
- Supersedes: the schema-key part of ADR-0004, the room addressing in ADR-0008
- Relates to: ADR-0002, ADR-0003, ADR-0007

## Context

The system exists to build many projects, and nothing in it has a notion of one.
`.oma` holds a single flat settings tree; the tracker's items float in one
undivided space. Two projects would silently share everything.

Separately, addressing is inconsistent. Items, comments and events already carry
minted IDs. Item types and statuses are addressed by human-chosen strings —
`bug`, `open` — which are stable and are not display names, but are still
strings a person picked, unique only by convention and only within whatever
space happens to contain them.

## Decision

### The project is the top scope

Everything stored belongs to a project, except a shared scope for what is
genuinely machine-wide — credentials, model defaults, the project registry
itself. Where both define the same setting, the project's value wins.

### Scoping is rooting, not parameterising

A project is a directory, and a component is handed a store already rooted in
it. `settings.Store` does not change: no method gains a project argument, no key
gains a prefix, and nothing has to remember to scope anything.

This is the strong version. A component holding project A's store cannot address
project B — not "must not", cannot. There is no argument to get wrong, because
isolation is a property of what the component was given rather than of how
carefully it behaves.

### Every entity has a minted ID, and IDs are readable

Names are display text: editable, duplicable, never an address. Every entity —
projects, item types, statuses, fields, options, and everything that already had
one — is addressed by an ID the system mints.

An ID is a primary key that a person can read. It carries a stem derived from
the name at creation plus a suffix that makes it unique:

    acme-site-4f7k          project
    bug-9c2x                item type
    in-review-x81p          status

The stem is **frozen at creation and will go stale**. Renaming "ACME Website" to
"Contoso" leaves `acme-site-4f7k` exactly as it is. That staleness is the point:
a stem that followed the name would be a name, and addressing by it would be the
thing this decision exists to prevent. The stem is a debugging convenience — it
keeps `.oma` browsable and a log line legible — and nothing may parse an ID or
infer anything from it.

### The tracker's schema keys become IDs

`TypeKey`, `StatusKey`, `FieldKey` and `OptionKey` become minted IDs, and the
strings that occupy them today become display names. This supersedes ADR-0004,
which made them stable human-chosen keys.

What ADR-0004 got right is untouched and is what makes this affordable:
`StatusCategory` is the axis every generic question is asked along, so no logic
anywhere reads a status key. Turning those keys opaque costs nothing at runtime
because nothing was reading them.

### Requests name the project in the path

    /projects/{project}/settings/{key}
    /projects/{project}/tracker/items/{item}

The project is part of a resource's identity, not ambient context, so it is
visible in a log line, a browser bar and a `curl`. A wrong project is a wrong
URL rather than a silent wrong answer.

Rooms gain the same scope, superseding ADR-0008's addressing: the `workspace`
room becomes `project:<id>`, and `item:` and `subtree:` rooms are project-scoped.

### A project lives anywhere, and the registry says where

A project's root defaults to `<workspace>/projects/<id>/`, and may be any
directory instead — so a project's tracker and settings can live inside the
repository they describe, versioned and moved with it.

    ~/.oma/
      shared/
        settings/            machine-wide
        projects/            the registry: one record per project
      projects/
        acme-site-4f7k/      default root
          settings/
          tracker/

**The registry is authoritative.** Every project has a record naming its ID, its
display name and its root, with the default filled in at creation. A directory
under `projects/` with no record is not a project, and a record is the only way
to discover one. Existence has one source of truth rather than two.

## Consequences

**Cross-project leakage stops being a class of bug.** No test can prove a
parameter is always passed correctly; rooting removes the parameter. This is the
main thing bought, and it is bought at the design level rather than by care.

**The registry is a single point of failure**, and deliberately so. Lose it and
projects still exist on disk but nothing can find them, which is the cost of
having one answer to "what projects are there" instead of a directory scan that
disagrees with a file.

**A root outside the workspace is not the workspace's to trust.** It can vanish,
sit on another filesystem, arrive through a git merge with conflict markers in
it, or be edited by hand while the server runs. Everything reached through the
registry needs the error handling that a directory you created yourself does
not.

**Readable stems will lie, and someone will parse one.** A stem is frozen while
the name it came from is not, so `acme-site-4f7k` will one day name a project
called something else. The mitigation is that nothing in the system may read a
stem — and that is a rule, which means it will need enforcing in review rather
than by the compiler.

**Migration is required and is not free.** The tracker's fixtures, the
conformance suite and the settings layout all assume a flat, unscoped world with
readable keys. None of it is deployed, so this is the cheapest this change will
ever be — which is the argument for making it now rather than the argument that
it is small.

**Two addressing styles are gone.** Everything is a minted ID, so there is one
rule to learn and one shape to validate, and the question "is this a key or an
ID" stops existing.
