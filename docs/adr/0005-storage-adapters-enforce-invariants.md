# ADR-0005: Storage adapters enforce the tracker's invariants

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api/internal/tracker`
- Relates to: ADR-0002, ADR-0004

## Context

ADR-0004 defines rules the tracker must uphold: values match their field's
kind, status moves follow the type's graph, the tree stays acyclic, a parent
cannot resolve before its descendants. Something has to enforce them, and there
are two places it could live.

A validating layer above storage is the usual answer: one implementation, every
adapter stays dumb. It has a real cost, though. A SQL database enforces
referential integrity, check constraints and triggers as its day job, far more
reliably than application code that has to remember to run. Putting the rules
above storage means writing that logic in Go and then _not_ using the database's
own guarantees — or writing it twice and hoping the two agree.

A filesystem is the opposite case. It has no constraint machinery at all, so
whatever it guarantees, it guarantees in code.

## Decision

Enforcement is part of the port contract, and each adapter satisfies it by
whatever means its backend makes natural.

- The filesystem adapter implements every check itself, in Go. `tracker.Validator`
  exists so it does not write the schema half by hand.
- The SQL adapter pushes the invariants into the schema — foreign keys, check
  constraints, a transition table, a trigger for the resolution gate — and
  writes no validation code of its own. Re-implementing in Go what the database
  already guarantees is the duplication this decision exists to avoid.

The invariants are the contract; the mechanism is an implementation detail. The
full list lives in `internal/tracker/contract.go`, next to the ports, because a
contract kept anywhere else goes stale.

## Consequences

**This is only safe with a shared conformance suite.** Rules distributed across
adapters are replaceable only if every adapter enforces them identically —
otherwise swapping storage changes behavior, which is exactly what ADR-0002
forbids. One suite, written against the ports, that every adapter must pass, is
what turns "both enforce this" from a claim into a test. The suite is therefore
not optional and not deferrable past the first adapter.

**Rules are stated once and implemented many times.** That is the accepted cost.
The mitigation is that the statement is executable — the conformance suite is
the specification, and an adapter that disagrees with it fails.

**A new adapter is expensive.** Not just reads and writes but every invariant,
proven. That is a deliberate brake: storage backends should be rare.

**Error identity matters more than usual.** A SQL constraint violation and a
Go-side check must surface as the same sentinel — `ErrUnresolvedDescendants` is
`ErrUnresolvedDescendants` whether it came from a trigger or an `if`. Adapters
translate at their own boundary, per ADR-0002.
