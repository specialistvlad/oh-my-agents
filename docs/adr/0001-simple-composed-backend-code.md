# ADR-0001: Backend code is simple, composed and aggregated

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api`

## Context

This system orchestrates agents that write software. The backend is where
coordination logic accretes, and coordination logic is exactly the kind that
grows a branch at a time until nobody can say what it does. Agents will also be
reading and editing this code, which rewards a codebase that is obvious over
one that is clever.

The convention has to exist before the code does. Retrofitting simplicity onto
a working system is a rewrite; requiring it from the first commit is free.

## Decision

All backend code must be very simple, and composed/aggregated.

Every unit does one obvious thing. Behaviour comes from wiring small units
together, never from one of them growing clever. When something gets hard to
follow, the fix is to split it and compose the pieces, not to add another
branch.

- Small functions, small files, small packages — one responsibility each.
- Build features by aggregating existing units. Reach for a new abstraction
  only after the composition is genuinely awkward.
- Prefer plain composition — a function handed exactly what it needs — over
  frameworks, reflection, or inheritance-shaped indirection.
- `cmd/` wires, `internal/` computes. Keep assembly at the edges.
- The boring version wins.

## Consequences

`make check` enforces a floor: no non-test file over 250 lines, no function
over 100 lines or 60 statements, no cyclomatic or cognitive complexity over 30.
These thresholds catch the failure; they do not define the standard. Code that
passes them can still be too clever, and review is expected to say so.

More files and more packages than a compact style would produce. That is the
intended trade: navigation cost is paid by tooling, comprehension cost is paid
by people.

The pressure to split is constant, and splitting badly produces indirection
without clarity. When a split does not make the caller easier to read, it is
the wrong split — undo it rather than keep it for the line count.
