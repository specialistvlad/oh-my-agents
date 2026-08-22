package projects

import "context"

// Store is the whole lifecycle, and what an implementation asserts against:
//
//	var _ projects.Store = (*projects.Registry)(nil)
//
// Consumers take the narrow part they need; only an adapter takes this.
//
// # What an implementation guarantees
//
// Create mints an id from the name, records the project, creates its root and
// marks it. An empty root means the default, `<workspace>/projects/<id>`.
//
// Rename changes display text only. The id and every path derived from it are
// untouched, which is the reason the two are separate.
//
// Repoint changes where the registry looks and **moves no files**. The caller
// relocates the directory and then says so. A copy across filesystems is
// neither atomic nor cheap, and a half-finished move leaves a project in two
// places, so a wrong path is corrected by pointing again.
//
// Remove deletes the record **and the root directory, wherever it lives**.
// Remove means remove. It refuses a root carrying no marker or a marker naming
// a different project, and refuses a filesystem root, a home directory or an
// ancestor of the workspace — so a hand-edited record or a mistyped repoint
// cannot become an arbitrary recursive delete. Nothing bounds it further:
// point a project at a directory holding other work and removal takes that
// too (ADR-0010).
//
// List returns every project, sorted by name, because the registry is the
// authority on what exists and a caller should not have to impose an order to
// get a stable one.
type Store interface {
	Create(ctx context.Context, n New) (Project, error)
	Get(ctx context.Context, id ID) (Project, error)
	List(ctx context.Context) ([]Project, error)
	Rename(ctx context.Context, id ID, name string) (Project, error)
	Repoint(ctx context.Context, id ID, root string) (Project, error)
	Remove(ctx context.Context, id ID) error
}

// New is the input to creating a project.
type New struct {
	Name string `json:"name"`
	// Root is where the project's data goes. Empty means the default.
	Root string `json:"root,omitempty"`
}
