# oh-my-agents

An orchestration system for agents: a software shop you can point at a project
and have it built from scratch. Work is tracked in a built-in task tracker and
driven from a web UI.

Early scaffolding — the services below run and are gated by CI, but the
orchestration itself is not built yet.

## Layout

| Path           | What it is                                          |
| -------------- | --------------------------------------------------- |
| `services/api` | Go backend. HTTP server, config, health/build-info. |
| `services/web` | React + Vite web UI (TanStack Router, MUI).         |

Each service owns its own toolchain and `make help`. The root `Makefile`
delegates to both.

## Getting started

```sh
make setup   # toolchains, dependencies, .env from .env.example
make start   # api on :39170, web on :39171
make check   # every gate both services enforce
```

## Configuration

One `.env` at the repo root, copied from `.env.example`. The api loads it
through its Makefile; the web UI points Vite's `envDir` at the same file and
sees the `VITE_`-prefixed values.

## Checks

`make check` runs, per service:

- **api** — `go vet`, `golangci-lint`, a 250-line-per-file limit, and unit,
  feature and component tests.
- **web** — build (which typechecks), Prettier, ESLint, and unit tests.

CI runs the same gates, minus the component tests, which need infrastructure a
runner does not have.
