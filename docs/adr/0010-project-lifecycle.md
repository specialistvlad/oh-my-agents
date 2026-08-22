# ADR-0010: Projects are created, renamed, re-pointed and removed, and every client sees it

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api`, `services/web`
- Relates to: ADR-0001, ADR-0008, ADR-0009

## Context

ADR-0009 made the project the scope everything hangs off, and nothing creates
one. It settled where a project's data lives and how it is addressed; it said
nothing about how a project comes into existence, changes, or stops existing.

That lifecycle is also the first thing more than one person watches at once. A
project list going stale is the plainest possible failure of a system whose
whole premise is that nothing polls.

## Decision

### The registry is a settings store, not a new one

A project record is a JSON document addressed by an id. That is exactly what
`settings.Store` already is, so the registry is one of those rooted in the
shared scope, with records at `projects/<id>`.

Nothing new is built: no second store, no second conformance suite, no second
set of guarantees to keep aligned. The registry inherits everything settings
already proved, including its lack of cross-process locking (ADR-0009).

### A project is an id, a name, and a root

The id is minted, readable and immutable (ADR-0009). The name is display text
and may be edited freely. The root is where the project's data lives.

**Create** mints the id from the name, writes the record, creates the root and
places a marker inside it. The root defaults to `<workspace>/projects/<id>` and
may be any directory.

**Rename** changes the name. The id and every path derived from it are
untouched, which is the point of separating them.

**Re-point** changes where the registry looks and **moves no files**. The user
relocates the directory and then says so. Moving data across filesystems is
neither atomic nor cheap, and a half-finished move leaves a project in two
places — so a wrong path is corrected by pointing again, not by a migration.

**List** reads the registry, which is authoritative: a directory with no record
is not a project.

**Remove deletes the record and the root directory, wherever it lives.** Remove
means remove. The safeguards below are what make that survivable rather than
qualifications on it.

### The marker is what makes removal safe to mean

Every project root holds a marker file naming the project it belongs to.

Removal refuses a root with no marker, or one naming a different project. So a
registry record edited by hand, or a path typed wrongly into a re-point, cannot
turn removal into an arbitrary recursive delete. Removal also refuses a root
that is a filesystem root, a home directory, or an ancestor of the workspace —
paths where a mistake is unrecoverable rather than merely annoying.

The marker is written on create and on re-point, so pointing a project at a
directory is the act that makes it a project root.

### Every change reaches every client

Project mutations announce on the bus from the store, exactly as settings do
(ADR-0008) — never from an edge, so HTTP and the socket are heard equally.

The room is `projects`, in the shared scope. It sits **above** any project,
which is why ADR-0009's project-scoped rooms do not fit it: a client watching
the list is not yet watching a project, and a client that just lost its project
still needs to hear that it is gone.

Three kinds: `project.created`, `project.changed`, `project.removed`. Each
carries the id, and creation and change carry the record, because a list is
small and a client that must fetch after every event is polling with extra
steps. Recovery is unchanged: on connect, and after any gap, fetch the list.

Both edges carry the whole lifecycle, as settings already do: HTTP for scripts
and anything that cannot hold a socket, the socket for anything that can.

## Consequences

**Removal is destructive and will one day delete something someone wanted.**
That is the accepted trade for a remove that means remove. The marker bounds
the blast radius to directories the system was told are projects, and nothing
bounds it further: point a project at a directory holding other work, and
removing the project takes that work with it. The guidance is that a project
root is a directory for the project and nothing else, and the API documentation
has to say so where someone will read it.

**Re-pointing and removal compose into the sharp edge.** Re-point marks a
directory as a project root; remove then deletes it. The two are individually
reasonable and together are how an accident happens, so re-point is the place
to be loud — it writes a marker into a directory the user may not have thought
of as ours.

**The registry is a single writer's story.** Two processes creating projects at
once can lose a record, because the settings store has no cross-process locking.
Unlike a lost setting, a lost record makes a directory full of work
undiscoverable, and it is the first thing file locking would be for.

**Events carry records, so they carry names.** A project name is user text and
now travels to every connected client. That is intended — it is what a list
needs — but it means the `projects` room is the first place where what one user
typed is shown to another, and the first that will need authorization when auth
arrives.

**Nothing else has to change to become project-aware.** Scoping is rooting
(ADR-0009), so a project handing out a store rooted at its own directory is the
entire mechanism. `settings.Store` gains no argument, and the tracker will not
either.
