# Architecture decision records

One file per decision, numbered in order, never rewritten — a decision that no
longer holds gets superseded by a later ADR rather than edited.

A **proposed** ADR states a decision that has not been taken: it lays out the
options and the questions blocking it, and becomes accepted when they are
answered. Nothing is built on a proposed ADR. Accepting one is the normal
lifecycle rather than a rewrite — the never-rewritten rule protects decisions
already taken.

| ADR                                                    | Decision                                                                               |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------- |
| [0001](0001-simple-composed-backend-code.md)           | Backend code is simple, composed and aggregated                                        |
| [0002](0002-layered-replaceable-implementations.md)    | The backend is layered so implementations are replaceable                              |
| [0003](0003-interfaces-and-objects-first-class.md)     | Interfaces and objects are first class, amended for Go                                 |
| [0004](0004-task-tracker-domain-model.md)              | The task tracker is one generic item type with a runtime schema _(superseded in part)_ |
| [0005](0005-storage-adapters-enforce-invariants.md)    | Storage adapters enforce the tracker's invariants                                      |
| [0006](0006-settings-store-on-the-filesystem.md)       | Settings are a filesystem key/value store rooted at `.oma` _(superseded in part)_      |
| [0007](0007-workspace-defaults-to-the-user-home.md)    | The `.oma` workspace lives in the user's home, and stays separate from `.env`          |
| [0008](0008-realtime-communication-over-websockets.md) | Clients talk over one bidirectional WebSocket, scoped by rooms _(superseded in part)_  |
| [0009](0009-projects-scope-everything.md)              | Everything is scoped to a project and addressed by a minted ID                         |
| [0010](0010-project-lifecycle.md)                      | Projects are created, renamed, re-pointed and removed, and every client sees it        |
| [0011](0011-three-column-workspace.md)                 | The UI is a three-column workspace built around state that arrives                     |
| [0012](0012-no-authentication.md)                      | There is no authentication, by design                                                  |
| [0013](0013-item-ordering.md)                          | Items carry one global rank, and reordering is not an edit                             |
| [0014](0014-editing-under-compare-and-swap.md)         | Editing keeps one version per item, and never loses a draft                            |
| [0015](0015-schema-editing.md)                         | Schema edits are granular, server-minted, and never silently lossy                     |
| [0016](0016-querying.md)                               | Query widens for fields and sorting, and stops before comparators                      |
| [0017](0017-boards-and-views.md)                       | A column is a status, a board is one type, and a view is local                         |
