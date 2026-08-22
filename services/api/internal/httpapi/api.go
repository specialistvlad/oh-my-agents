// Package httpapi assembles the HTTP surface.
//
// One place owns the URL space. Per-project resources nest inside
// /projects/{project}/ (ADR-0009), so no single package can be mounted at a
// prefix and own its own subtree — and having each register absolute patterns
// on a shared mux keeps the whole shape readable in one file rather than
// spread across whoever happens to mount whom.
package httpapi

import (
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projectshttp"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settingshttp"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/trackerhttp"
)

// Scopes resolves a project into whatever is stored inside it. One interface
// satisfies every per-project package, so the surface takes one dependency
// rather than one per resource.
type Scopes interface {
	settingshttp.Scopes
	trackerhttp.Scopes
}

// Deps are the stores the surface is built over.
type Deps struct {
	Projects projectshttp.Store
	Scopes   Scopes
}

// New returns the API handler.
func New(d Deps) http.Handler {
	mux := http.NewServeMux()
	projectshttp.Register(mux, d.Projects)
	settingshttp.Register(mux, d.Scopes)
	trackerhttp.Register(mux, d.Scopes)
	return mux
}
