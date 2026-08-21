# ADR-0007: The `.oma` workspace lives in the user's home, and stays separate from `.env`

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/api`
- Supersedes: the default-location part of ADR-0006
- Relates to: ADR-0002, ADR-0006

## Context

ADR-0006 put the workspace at `.oma` in the process's working directory. That
turned out to be the wrong default for two reasons.

It fragments. Start the server from `services/api` instead of the repo root and
you silently get a second workspace, with different settings and no warning.
The failure is invisible until someone wonders why a setting they just changed
did nothing.

And it misreads what `.oma` is for. The folder holds what the _running system_
accumulates — settings today, and more later, the way a `.claude` folder holds
several kinds of thing. That belongs to the person operating the machine, not
to whichever directory a shell happened to be in.

Separately, there are now two distinct sources of configuration, and it is
worth being explicit about which is which before they blur.

## Decision

**The workspace defaults to `.oma` in the user's home directory.** One
workspace per user per machine, found the same way no matter where a process
starts.

**`OMA_HOME` overrides it**, and appears in `.env.example` with the default
spelled out. A leading `~` is expanded by the application rather than left to a
shell, because the value often arrives from a container environment where
nothing has expanded it. Relative paths are resolved to absolute at
construction, so a later `chdir` cannot move the workspace out from under a
running process.

**The two configuration sources stay separate.**

|            | `.env` / environment | `.oma` workspace         |
| ---------- | -------------------- | ------------------------ |
| Owner      | `internal/config`    | `internal/settings`      |
| Read       | once, at boot        | continuously, at runtime |
| Written by | deployment           | the running system       |
| Mutable    | no                   | yes                      |

`internal/config` no longer imports `internal/settings`. It reads `OMA_HOME`
verbatim and passes the string through; the workspace owns its own default,
its own tilde expansion, and its own idea of where it lives. The string is the
only thing that crosses between them, in one direction.

**A store that cannot resolve its root fails at construction.** `NewFS` returns
an error rather than deferring the problem to the first write, which is the
difference between a failed boot and a puzzling runtime error hours later.

## Consequences

Settings now survive between projects and outlive any single checkout, which is
what makes them settings rather than project files. A workspace that should be
project-scoped needs `OMA_HOME` set deliberately — there is no automatic
per-project workspace any more, and no discovery by walking up parent
directories.

The home directory has to be resolvable. In a container running as a user with
no home entry, `os.UserHomeDir` fails and the process refuses to start unless
`OMA_HOME` is set. That is the intended trade: an explicit failure beats
writing settings somewhere nobody will look.

`.oma` remains gitignored, which now matters less, since the default is no
longer inside a repository at all.
