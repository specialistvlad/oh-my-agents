# ADR-0003: Interfaces and objects are first class, amended for Go

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api`
- Relates to: ADR-0002

## Context

ADR-0002 requires ports and adapters but does not say how types should be
designed on either side of them. Left unstated, that gap gets filled two ways,
both bad: procedural code over anonymous maps that has no seams to make
replaceable, or Java transplanted into Go — base types, `IFoo` interfaces,
getter ceremony, a DI container — which fights the language and produces
indirection nobody can follow.

## Decision

Design in interfaces and objects first. Behaviour belongs to a type that owns
its state, and every collaborator is reached through an interface that names a
capability. Free functions over anonymous data are the exception.

Where Go's idioms differ from the classical object-oriented playbook, the Go
amendments win:

- No inheritance. Compose and embed instead — a type gets behaviour by holding
  the thing that has it, never by extending a base class.
- Interfaces stay small: one to three methods, `io.Reader`-shaped. A wide
  interface is a missing decomposition, not a rich one.
- Accept interfaces, return concrete types. The caller decides what it needs;
  the constructor hands back something specific.
- Name the interface for the capability (`TaskStore`, `Clock`) and the
  implementation for the technology (`fsTaskStore`, `postgresTaskStore`). No
  `I` prefixes, no `Impl` suffixes.
- No getter/setter ceremony. If a field is just data, export it.
- Wire by hand in `cmd/`. No DI container, no service locator, no reflection
  standing in for a constructor.

## Consequences

The interface rule and ADR-0002's port rule are the same rule seen twice: an
interface defined where it is consumed, kept to the methods that consumer
calls, is already a port. Small interfaces are also what make the required
in-memory fakes cheap to write — a three-method port has a fake worth keeping,
a fifteen-method one does not.

Hand-wiring means `cmd/` grows as the system does, and every dependency is
visible there. That is the intent: the wiring is the architecture, and it
should be readable in one file rather than inferred from annotations.

Some duplication is expected, since a capability consumed by two packages is
declared by both rather than shared from one place. Go's structural typing
means the implementation satisfies both without knowing either exists, and two
narrow declarations beat one shared interface that widens to serve everyone.
