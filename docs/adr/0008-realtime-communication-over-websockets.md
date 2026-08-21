# ADR-0008: Clients talk over one bidirectional WebSocket, scoped by rooms

- Status: Accepted; room addressing is superseded by ADR-0009
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

`Bus` ships with **two implementations from the start**, held to one
conformance suite per ADR-0005:

- **in-memory** — channels, one process. The default.
- **Valkey** — the BSD-licensed fork of Redis, so the protocol and every client
  library still apply without the licence. On port 39172, beside the api on
  39170 and the web app on 39171, so a Valkey or Redis already running for
  something else cannot be joined by accident.

`VALKEY_URL` unset selects the in-memory bus. **`make start` installs and runs
nothing**: clone the repo, start the server, open the app, and realtime works.
Valkey is what you turn on when you run more than one process, and turning it
on is a config change and no code at all.

This is a standing rule, not a concession for this one dependency: anything
external the system later grows — a work queue, a scheduler — arrives with an
in-memory implementation beside it, and the default configuration keeps
requiring nothing.

**The bus carries notification, not truth.** A published message is a
low-latency hint that activity exists; correctness comes from `Seq`. A dropped
message costs latency, not consistency, because the next one — or a reconnect —
exposes the gap and the client backfills from the store. This is what makes
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

**Two structurally different implementations at last.** Until now ADR-0005's
conformance suite has only ever had memory-vs-memory to compare, which proves
little. Channels versus a network round trip is a genuine difference, and it is
where the suite starts earning its keep.

**Valkey is opt-in, and the switch has to be exercised.** A path only taken in
production is a path that breaks in production, so the Valkey bus needs a test
run in CI against a real instance even though no default configuration uses it.

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
queue; on overflow the server drops its backlog and sends one `resync` frame.
The client backfills from `Seq` — the same recovery as any other gap, which is
why there is only one recovery path to get right.
