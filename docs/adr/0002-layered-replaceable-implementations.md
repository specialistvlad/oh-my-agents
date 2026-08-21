# ADR-0002: The backend is layered so implementations are replaceable

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api`

## Context

The backend has no persistence yet — deliberately, since the storage question
is not answerable before the task tracker's shape is known. Committing to a
database now would embed that guess in every caller, and unpicking it later
touches everything.

There is also a testing motive. `tests/feature` is meant to run whole scenarios
in CI with no infrastructure, which is only possible if the system can be
assembled against something other than its production dependencies.

## Decision

All code must be multilayered, so one implementation can be swapped for another
with no changes upwards.

The reference case: replacing the database with a filesystem implementation —
plain files on disk, same behaviour — must be a change in `cmd/` and nowhere
else. Every caller above it is untouched, and untouchable.

- Logic depends on interfaces, never on implementations. Dependencies point one
  way: `cmd/` wires, logic calls a port, an adapter satisfies it. Nothing above
  an adapter may import it.
- The consumer owns the interface. Define it in the package that needs it, and
  keep it to the methods that package actually calls.
- Only domain types cross a port. Driver and framework types stop at the
  adapter, and so do their errors — the adapter translates both.
- Every port gets two implementations: the real one and an in-memory fake.

## Consequences

The seam is verified rather than asserted. `tests/feature` runs whole scenarios
on the fakes, so a port that has quietly become unswappable fails a test
instead of being discovered during a migration.

Ports stay narrow, because the filesystem implementation is the yardstick. If
files on disk cannot honour a guarantee, that guarantee does not belong in the
port — which rules out leaking a query language into a method name, a
transaction the caller must remember to open, or an ordering only one backend
happens to provide. Leaks are usually semantic like these, not structural, so
an import-graph check is not sufficient review.

Capability the production backend has and the port does not expose is
capability the system cannot use. That cost is accepted: a feature that
genuinely needs it can widen the port deliberately, having first answered how
the fake will honour it.

The same rule is expected to apply to the web app's data access when that
lands, but that has not been decided here.
