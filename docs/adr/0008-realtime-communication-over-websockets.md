# ADR-0008: Clients talk over one bidirectional WebSocket, scoped by rooms

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api`, `services/web`
- Relates to: ADR-0002, ADR-0004, ADR-0005

## Context

Several clients watch the same work at once — people in a browser, agents
reporting progress — and everything has to stay current without anyone asking
whether it has. Polling is ruled out.

Half the machinery already exists. ADR-0004 gave every change an entry in an
append-only feed with a monotonic `Seq`, which is exactly what a client needs to
resume from where it left off. What is missing is that `EventReader.Events` is a
**pull** interface: the only way to learn something happened is to ask. A socket
layer built on it would poll internally and we would have moved the polling
rather than removed it.

So the transport is not the whole decision. Reactive delivery needs a push seam
on the store, and push across more than one process needs something between
them.

## Decision

### One socket, bidirectional

A client holds a single WebSocket. It carries events out and commands in —
joins, leaves, resumes, and mutations. There is no second connection and no
request the client has to remember to repeat.

The socket is an **edge**, not an application. Every command it accepts is
executed through the same `tracker` ports the HTTP surface uses. Neither edge
gets its own copy of a rule, or ADR-0005's warning applies to the two of them
instead of to two adapters.

Bidirectional buys fewer round trips and costs three things a request/response
protocol gives away free. Each is paid for explicitly:

- **Correlation.** Every command carries a client-generated `id`. Every reply —
  `ack` or `error` — carries the same one. A client with two writes in flight
  can tell which came back.
- **Idempotency.** Every command carries a client-generated key. A reconnecting
  client replays what it never saw acknowledged, and the server recognizes the
  key rather than applying the write twice. Without this, a dropped
  acknowledgement silently duplicates work.
- **Conflicts.** A stale `Version` comes back as an `error` frame carrying the
  current one, so the client re-reads and retries rather than guessing what
  happened.

### The event log stays the source of truth

The socket delivers; it does not decide. Every frame a client receives
corresponds to an entry in the feed and carries its `Seq`.

That is what makes reconnection cheap. A client resumes by naming the last `Seq`
it handled per room; the server backfills from the store, then switches to live
delivery. A gap in `Seq` is detectable by the client, and the answer to a gap is
always the same: backfill from the store.

### A push port on the store

`tracker` gains a subscription port alongside `EventReader` — read it as
`Subscribe(ctx, since Seq) (<-chan Event, error)`. An in-process store fans out
on write. This is the seam that makes the whole design push rather than pull,
and it is where ADR-0002's layering first gets genuinely hard: a filesystem
store can notify subscribers inside its own process, and cannot tell another
process anything at all.

### A broker from the start

Processes learn about each other's writes through **Valkey** pub/sub — the
BSD-licensed fork of Redis, so the protocol and every client library still
apply, without the licence. It listens on a port of its own (39172, beside the
api on 39170 and the web app on 39171) so a Valkey or Redis already running for
something else cannot be joined by accident.

**The broker carries notification, not truth.** A published message is a
low-latency hint that activity exists; correctness comes from `Seq`. A dropped
message costs latency, not consistency, because the next one — or a reconnect —
exposes the gap and the client backfills from the store. This is what makes
fire-and-forget pub/sub safe here, and it is the reason not to reach for a
broker with delivery guarantees.

### Rooms address the tree

Three room kinds, matching what the tracker already is:

| Room           | Carries                        |
| -------------- | ------------------------------ |
| `workspace`    | all activity                   |
| `item:<id>`    | that item only                 |
| `subtree:<id>` | that item and every descendant |

Opening an epic is one join. Membership needs no query engine: the rooms an
event belongs to are its item's ancestors, walked through `SubtreeReader`.

A reparenting belongs to **two** ancestor sets — the one it left and the one it
joined — and is published to both, or the branch it left never learns it is
gone. Deletion is computed before the item disappears, for the same reason.

## Consequences

**Valkey becomes the first required infrastructure.** A project that needed
nothing to run now needs something, in development as well as deployment. That
is the price of choosing horizontal scale before it is needed, taken knowingly.
The non-default port keeps it from colliding with anything already on the
machine, which matters more here than usual: joining a stranger's Redis would
look like events silently going missing.

**Every event costs an ancestor walk** to decide its rooms. Cheap on a shallow
tree, linear in depth, and depth is unbounded by ADR-0004. If it becomes hot,
the answer is a cached ancestor path per item — not a change to the room model.

**Rooms are addressing, not a guarantee.** They scope what a client asks for,
and nothing yet stops a client asking for a room it should not see, because
there is no authentication anywhere in this service. "Everything they need, not
more" is delivered as intent now and as a property only once auth exists.
Authentication is a prerequisite of exposing this beyond a trusted network, and
is the largest open risk in this decision.

**A broker fixes fan-out, not shared storage.** Processes can now tell each
other what happened, but the stores underneath still have no cross-process
locking — settings says so explicitly, and the tracker's filesystem adapter does
not exist yet. Running two processes against one workspace remains unsafe for
reasons this ADR does not address.

**Two edges must be kept honest.** HTTP stays, for scripts, for anything that
cannot hold a socket, and because a request/response API is easier to debug.
Every new capability now has two front doors, and the discipline that keeps that
from becoming two behaviours is that both call the same ports and neither
validates anything itself.

**Backpressure is a real case, not an edge case.** A slow client gets a bounded
queue; on overflow the server drops its backlog and sends one `resync` frame.
The client backfills from `Seq` — the same recovery as any other gap, which is
why there is only one recovery path to get right.
