# api

Go backend for oh-my-agents.

## Core principle

**All backend code must be very simple, and composed/aggregated.**

Every unit does one obvious thing. Behaviour comes from wiring small units
together — never from one of them growing clever. When something gets hard to
follow, the fix is to split it and compose the pieces, not to add another
branch.

In practice:

- Small functions, small files, small packages — one responsibility each.
- Build features by aggregating existing units; reach for a new abstraction
  only after the composition is genuinely awkward.
- Prefer plain composition — a function handed exactly what it needs — over
  frameworks, reflection, or inheritance-shaped indirection.
- `cmd/` wires, `internal/` computes. Keep assembly at the edges.
- The boring version wins.

`make check` enforces a floor for this: no non-test file over 250 lines, no
function over 100 lines or 60 statements, no cyclomatic or cognitive
complexity over 30. Passing those is not the same as being simple — they catch
the failure, they do not define the standard.

## Layout

| Path                  | What it is                                          |
| --------------------- | --------------------------------------------------- |
| `cmd/server`          | Entrypoint. Reads config, starts the server, waits. |
| `internal/config`     | The only place env is read; operational defaults.   |
| `internal/httpserver` | Listener, routes, graceful shutdown.                |
| `tests/feature`       | Whole scenarios against in-memory fakes.            |
| `tests/component`     | Tests that need real infrastructure.                |

`make help` lists the targets.
