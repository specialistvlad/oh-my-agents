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
)

// Deps are the stores the surface is built over.
type Deps struct {
	Projects projectshttp.Store
	Scopes   settingshttp.Scopes
}

// New returns the API handler.
func New(d Deps) http.Handler {
	mux := http.NewServeMux()
	projectshttp.Register(mux, d.Projects)
	settingshttp.Register(mux, d.Scopes)
	return mux
}
