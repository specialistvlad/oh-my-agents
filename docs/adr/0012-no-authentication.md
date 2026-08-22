# ADR-0012: There is no authentication, by design

- Status: Accepted
- Date: 2026-08-21
- Scope: whole system
- Supersedes: the "auth is a prerequisite" framing in ADR-0006 and ADR-0008
- Relates to: ADR-0004, ADR-0009, ADR-0010

## Context

Every decision since ADR-0006 has described the absence of authentication as
something to be fixed: a prerequisite, an open risk, the largest thing
outstanding. That framing is wrong, and leaving it in place has a cost of its
own — it defers questions to a design that is not coming, and it makes every
consequence section end on the same unfinished note.

The system is a workspace on somebody's machine. It stores its state in a
directory in their home or their repository, and the people and agents using it
are already the people and agents with access to that directory.

## Decision

**There is no authentication, and none is planned.** The system trusts whoever
can reach it.

Three things follow directly, and they are the decision rather than
consequences of it:

**Every actor is self-declared.** An [`ActorRef`] is who a caller says they
are. Humans and agents alike name themselves, and nothing checks. The activity
feed records claims.

**Rooms are addressing, not access control.** A client may join any room it
names. ADR-0008's "everything they need, not more" describes what a client asks
for, and this settles what it never was: an enforced boundary.

**The server binds every interface.** It listens on `:39170`, not
`127.0.0.1:39170`, so it is reachable from the network it is on without
configuring anything.

## Consequences

**The network is the boundary, and it is the only one.** Anyone who can route
to the port can read every project, write any setting, and `DELETE` a project —
which removes its root directory and everything in it (ADR-0010). On a shared
network that is everyone on it. This is the decision's real cost, stated once
here rather than repeated as a warning at every edge.

**The audit trail records claims, not identities.** `ActorRef` is still worth
having: it is how a person reconstructs what happened, and how one agent's work
is told from another's. It is not evidence, and nothing built on it may treat
it as such.

**Multi-tenancy is foreclosed, not deferred.** There is no place in the design
to put a tenant, because projects are scoped by rooting (ADR-0009) and rooting
has no notion of who is asking. A hosted, shared deployment is a different
system, not a later version of this one.

**A great deal of work does not exist.** No sessions, no tokens, no login, no
permission model, no per-actor filtering of rooms or queries, and no decision
about how an agent authenticates differently from a person. That absence is
what the decision buys, and it is substantial.

**Reversing this is a new decision, superseding this one.** It would touch
every edge, the room model, and the meaning of `ActorRef` — so it is worth
knowing in advance that it is not a feature to be added but a design to be
replaced.
