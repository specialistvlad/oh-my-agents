# ADR-0008: Clients talk over one bidirectional WebSocket, scoped by rooms

- Status: Accepted; room addressing is superseded by ADR-0009, and the auth
  framing by ADR-0012
- Date: 2026-08-21
- Scope: `services/api`, `services/web`
- Relates to: ADR-0002, ADR-0004, ADR-0005

## Context

Several clients watch the same work at once — people in a browser, agents
reporting progress — and everything has to stay current without anyone asking
whether it has. Polling is ruled out.

What is missing is not a way to ask. `EventReader.Events` is a **pull**
interface: the only way to learn that something happened is to go and look. A
socket layer built on that would poll internally, and we would have moved the
polling rather than removed it.

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

### State on connect, updates thereafter

A client fetches the current state of what it is showing, once, and then lives
on updates. There is no replay, no resume token, and no history to reconstruct:
the socket never redelivers what happened before a client cared, and a client
never refetches something it is already watching.

**Join first, then fetch.** That order is the whole correctness argument. A
change during the fetch either lands in the state that comes back, or arrives
as an event afterwards. Fetching first would leave a window between the read
and the subscription in which a change is lost and nobody knows it.

Applying an update the fetch already reflects therefore has to be harmless.
Items carry a `Version` (ADR-0004), so a client discards an event describing a
version it already holds; anything without a version has to be idempotent.

Recovery is the same move, not a special one. A gap in `Seq`, or a `resync`
frame, means only "you missed something" — the answer is to fetch the current
state of what is on screen again, never to ask what happened. One path, used
on connect and on recovery alike.

The tracker's event feed (ADR-0004) stays what it always was: an activity
record for people to read. It is not a synchronisation mechanism, and nothing
here depends on it.

### A push port on the store

`tracker` gains a subscription port alongside `EventReader` — read it as
`Subscribe(ctx, since Seq) (<-chan Event, error)`. An in-process store fans out
on write. This is the seam that makes the whole design push rather than pull,
and it is where ADR-0002's layering first gets genuinely hard: a filesystem
store can notify subscribers inside its own process, and cannot tell another
process anything at all.

### The whole path is layered

Four seams, each a port, none naming a technology above it:

| Seam          | Question it answers                       | Implementations     |
| ------------- | ----------------------------------------- | ------------------- |
| `EventSource` | what just happened in this process        | the store, on write |
| `Bus`         | what just happened in the other processes | in-memory, Valkey   |
| `Hub`         | which connections care                    | rooms               |
| `Transport`   | how bytes reach a client                  | WebSocket           |

`Bus` is the one that would otherwise leak infrastructure upwards, so it is
kept narrow — read it as `Publish(ctx, Event) error` and
`Subscribe(ctx) (<-chan Event, error)`. Nothing above it knows whether the
process next door is reached through a channel or a socket.

### The default needs nothing installed

`Bus` has **one implementation: in-memory channels, one process.** It is the
default, and while the stores underneath still have no cross-process locking it
is also the only configuration that is actually sound. **`make start` installs
and runs nothing** — clone, start, open the app, and realtime works.

**Valkey is the second implementation and is deliberately not built.** Writing
it now would mean maintaining and testing a path nothing uses, against an
infrastructure dependency nothing needs, to enable a multi-process
configuration that is unsafe for other reasons. What exists now is the port,
which is what makes adding it later an implementation and a config change
rather than a rewrite. `VALKEY_URL` is reserved for it, and the conformance
suite is written so a second implementation is held to it the day it appears.

When it arrives it will be Valkey — the BSD-licensed fork of Redis, so the
protocol and every client library still apply without the licence — on port
39172, beside the api on 39170 and the web app on 39171, so a Valkey or Redis
already running for something else cannot be joined by accident.

This is a standing rule, not a concession for this one dependency: anything
external the system later grows — a work queue, a scheduler — arrives with an
in-memory implementation beside it, and the default configuration keeps
requiring nothing.

**The bus carries notification, not truth.** A published message is a
low-latency hint that activity exists; correctness comes from `Seq`. A dropped
message costs latency, not consistency, because the next one — or a reconnect —
exposes the gap and the client refetches what it is showing. This is what makes
fire-and-forget pub/sub safe, and the reason not to reach for a broker with
delivery guarantees.

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

**Nothing is required to run, and that is load-bearing.** A contributor gets a
working realtime system from a clone, and every test exercises the real bus
rather than a stub. The cost is that the in-memory bus is a real implementation
to be maintained and not a toy: it has to honour ordering and backpressure the
same way Valkey does, or passing tests against it will mean nothing.

**The conformance suite is written for an implementation that does not exist.**
Until Valkey lands it only ever has memory to compare against itself, which
proves little. That is accepted: the suite's job today is to write down what a
bus must do, so the second one is measured against a definition rather than
against whatever the first happened to do.

**A port with one implementation is an untested claim.** Nothing proves the seam
is really replaceable until something else sits behind it, and the usual way
that goes wrong is an interface shaped around its only implementation. The
narrowness of `Publish`/`Subscribe` is the hedge, not a guarantee.

**Every event costs an ancestor walk** to decide its rooms. Cheap on a shallow
tree, linear in depth, and depth is unbounded by ADR-0004. If it becomes hot,
the answer is a cached ancestor path per item — not a change to the room model.

**Rooms are addressing, not a guarantee.** They scope what a client asks for,
and nothing yet stops a client asking for a room it should not see, because
there is no authentication anywhere in this service. "Everything they need, not
more" is delivered as intent now and as a property only once auth exists.
Authentication is a prerequisite of exposing this beyond a trusted network, and
is the largest open risk in this decision.

**Turning Valkey on fixes fan-out, not shared storage.** It lets processes tell
each other what happened; it does nothing about the stores underneath, which
have no cross-process locking — settings says so explicitly, and the tracker's
filesystem adapter does not exist yet. So the multi-process configuration this
bus enables is not yet safe for reasons this ADR does not address, and the
single-process default is the only one currently sound end to end.

**Two edges must be kept honest.** HTTP stays, for scripts, for anything that
cannot hold a socket, and because a request/response API is easier to debug.
Every new capability now has two front doors, and the discipline that keeps that
from becoming two behaviours is that both call the same ports and neither
validates anything itself.

**Backpressure is a real case, not an edge case.** A slow client gets a bounded
queue; on overflow the server drops that backlog and sends one `resync` frame.
Dropping the backlog is not a compromise: the client is about to refetch, so
everything queued is already superseded. It then refetches what it is showing —
the same move as on connect, which is why there is only one recovery path to
get right.
