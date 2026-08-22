# ADR-0006: Settings are a filesystem key/value store rooted at `.oma`

- Status: Accepted; the default location is superseded by ADR-0007, and the
  auth framing by ADR-0012
- Date: 2026-08-21
- Scope: `services/api/internal/settings`
- Relates to: ADR-0002, ADR-0003, ADR-0005

## Context

The system needs configuration it can change while running — preferences,
defaults, whatever an operator adjusts — and that is a different thing from
deploy-time configuration. Env vars are read once at boot by `internal/config`
and never change; settings are written by the running system and read back
later.

There is no database, deliberately (ADR-0002). Something has to hold them now,
and it has to be replaceable when that answer changes.

## Decision

**A key/value store of JSON documents**, addressed by slash-separated keys —
`agent/model`, `web/theme`. Keys are validated by a narrow grammar because they
become filesystem paths.

**The port is byte-oriented and three methods wide.** `Reader`, `Writer` and
`Lister` deal in `Document`, not in application types. Typed access is composed
on top as generic free functions, `Read[T]` and `Write[T]`, so the interface an
adapter implements never widens no matter how many types callers store through
it. This is ADR-0003's small-interface rule taken seriously: the ergonomics go
beside the interface, not inside it.

**The default root is `.oma` in the process's working directory**, overridable
with `OMA_HOME`. One file per key at `<root>/settings/<key>.json`, which makes
settings readable and editable by hand with the application stopped. That is a
feature, not an accident of the implementation.

**Writes are atomic** — temp file in the destination directory, `fsync`,
rename — so a reader sees the old document or the new one and never a partial
file.

**Two implementations, one suite.** `FS` and `Memory`, both held to the same
guarantees by `settingstest`. Per ADR-0005 each enforces the rules itself; the
suite is what proves they agree.

## Consequences

Settings are inspectable with `cat` and editable with `$EDITOR`. Debugging and
seeding both get easier, and an operator can fix a bad value without the
service running.

`.oma` is resolved relative to the working directory, so where a process is
started changes which settings it sees. The boot log prints the absolute path
it settled on, because that will otherwise be confusing exactly once per person.

**No cross-process locking.** A mutex serializes one process against itself.
Two processes sharing a root can interleave writes and lose one. That is
acceptable while a single API process owns the directory, and is the first
thing to fix when that stops being true.

**The HTTP surface has no authentication**, because nothing in this service has
any yet. It writes to disk on an unauthenticated `PUT`, so until auth exists it
must not be reachable from outside a trusted network. This is the largest open
risk in the change.

Deleting the last key in a namespace leaves an empty directory behind. Pruning
races with concurrent writes, so the directories stay; `Keys` ignores them.
