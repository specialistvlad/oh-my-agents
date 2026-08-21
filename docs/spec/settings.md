# Spec: settings store

Implementation spec for `services/api/internal/settings`. The decision is
[ADR-0006](../adr/0006-settings-store-on-the-filesystem.md).

Status: **implemented, wired and running.**

## Model

A setting is a JSON `Document` addressed by a `Key`.

`Key` is one or more segments joined by `/`. Each segment is alphanumerics,
underscore and hyphen, optionally dotted — `agent.model`, `web/theme`,
`a/b/c/d`. Rejected outright: empty, `.`, `..`, leading or trailing dots,
empty segments, anything over 512 bytes, and every character outside the
grammar. Keys become paths, so the grammar is the containment.

`Document` must be valid, non-empty JSON.

## Ports

| Port     | Methods                       |
| -------- | ----------------------------- |
| `Reader` | `Get`                         |
| `Writer` | `Set`, `Delete`               |
| `Lister` | `Keys` — sorted               |
| `Store`  | all three; conformance target |

Typed access composes on top rather than widening the port:

```go
prefs, err := settings.Read[AgentPrefs](ctx, store, "agent/prefs")
prefs, err := settings.ReadOr(ctx, store, "agent/prefs", defaults)
err := settings.Write(ctx, store, "agent/prefs", prefs)
```

`ReadOr` falls back only on absence. A decode failure or an unreadable store is
still an error, because falling back on those hides a real problem.

## Guarantees

Every implementation, enforced by itself per ADR-0005 and verified by
`settingstest`:

- Invalid keys → `ErrInvalidKey`; invalid documents → `ErrInvalidDocument`.
- `Get` and `Delete` on an absent key → `ErrNotFound`.
- `Set` overwrites silently and is atomic.
- Documents are copied in both directions; a caller cannot reach the store's
  memory through a slice it was handed.
- `Keys` is sorted.
- Safe for concurrent use within one process.

## Implementations

**`FS`** — one file per key at `<root>/settings/<key>.json`. The root is the
`.oma` workspace: `~/.oma` by default, `OMA_HOME` to override, `~` expanded by
the app and relative paths made absolute at construction (ADR-0007).
Construction and reads touch nothing; the directory appears on first write.
Writes go to a temp file in the destination directory, are fsynced, then
renamed.

`.oma` is the workspace, not the settings store. Settings occupy
`<root>/settings/`, leaving the root free for whatever else the running system
needs to keep.

**`Memory`** — a map. The fake tests build on.

## HTTP

Mounted at `/settings/` by `cmd/server`.

| Method   | Path        | Result                            |
| -------- | ----------- | --------------------------------- |
| `GET`    | `/`         | `{"keys":[...]}`, always an array |
| `GET`    | `/{key...}` | the raw document, or 404          |
| `PUT`    | `/{key...}` | 204; 400 on a bad key or document |
| `DELETE` | `/{key...}` | 204, or 404                       |

Bodies are capped at 1 MiB; over the cap is 413, unreadable is 400. The
handler declares its own narrow `Store`
interface and names no storage technology, so it serves `FS` and `Memory`
identically — which the tests assert.

**Traversal is refused twice.** `ServeMux` normalizes a literal `/../` away
before routing; a percent-encoded `%2e%2e` survives that and is caught by the
key grammar. Neither layer alone is sufficient, and both are tested — including
one test that checks the filesystem directly to prove no write lands outside
the root.

**There is no authentication.** See ADR-0006.

## Known limits

- No cross-process locking.
- Deleting the last key in a namespace leaves an empty directory.
- The home directory must be resolvable, or `OMA_HOME` must be set; otherwise
  the process refuses to start. The boot log prints the absolute path it
  resolved.

## Next

1. Authentication before this is reachable from anywhere untrusted.
2. File locking if a second process ever shares a root.
3. Typed, named setting definitions with defaults and validation, once there
   are real settings to define.
