# api

Go backend for oh-my-agents.

## Core principles

Three rules govern everything here. The reasoning and the fine print are in the
ADRs; this is the short form.

1. **Simple, composed and aggregated** — [ADR-0001](../../docs/adr/0001-simple-composed-backend-code.md).
   Every unit does one obvious thing, and behaviour comes from wiring small
   units together. When something gets hard to follow, split it and compose;
   never add another branch.

2. **Layered, so any implementation is replaceable** — [ADR-0002](../../docs/adr/0002-layered-replaceable-implementations.md).
   Swapping the database for plain files on disk must be a change in `cmd/` and
   nowhere else. Logic depends on interfaces the consumer owns; only domain
   types cross a port; every port has a real implementation and an in-memory
   fake.

3. **Interfaces and objects are first class, amended for Go** — [ADR-0003](../../docs/adr/0003-interfaces-and-objects-first-class.md).
   Behaviour belongs to a type that owns its state. No inheritance, small
   interfaces, accept interfaces and return concrete types, wire by hand in
   `cmd/`.

`make check` enforces a floor for the first rule: no non-test file over 250
lines, no function over 100 lines or 60 statements, no complexity over 30.
Passing it is not the same as being simple.

## Layout

| Path                      | What it is                                           |
| ------------------------- | ---------------------------------------------------- |
| `cmd/server`              | Entrypoint. Reads config, starts the server, waits.  |
| `internal/config`         | The only place env is read; operational defaults.    |
| `internal/httpserver`     | Listener, routes, mounts, graceful shutdown.         |
| `internal/settings`       | Runtime settings, on the filesystem under `.oma`.    |
| `internal/settingshttp`   | The settings store over HTTP.                        |
| `internal/bus`            | Fan-out between processes. In-memory; Valkey later.  |
| `internal/idempotency`    | Remembers commands, so a replay does not repeat one. |
| `internal/settingsbus`    | A settings store that announces what it writes.      |
| `internal/projects`       | Project lifecycle. Its registry is a settings store. |
| `internal/projectsbus`    | A project store that announces what it changes.      |
| `internal/projectshttp`   | The project lifecycle over HTTP.                     |
| `internal/scopes`         | Hands out stores already rooted in a project.        |
| `internal/rooms`          | The realtime room names, spelled once.               |
| `internal/httpapi`        | Assembles the HTTP URL space.                        |
| `internal/realtime`       | Rooms and per-connection queues.                     |
| `internal/realtimews`     | The realtime hub over a WebSocket.                   |
| `internal/tracker`        | Task tracker domain model, ports and validation.     |
| `…/tracker/store`         | The tracker's rules, over a persistence port.        |
| `…/tracker/fs`            | File persistence for that store.                     |
| `internal/trackerhttp`    | A project's tracker over HTTP.                       |
| `…/tracker/memory`        | In-memory store. Enforces every invariant itself.    |
| `…/tracker/trackertest`   | Conformance suite every store must pass.             |
| `…/settings/settingstest` | The same, for settings stores.                       |
| `tests/feature`           | Whole scenarios against in-memory fakes.             |
| `tests/component`         | Tests that need real infrastructure.                 |

`make help` lists the targets.
