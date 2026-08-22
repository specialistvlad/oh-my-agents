// Package trackerhttp exposes a project's tracker over HTTP.
//
// The other edge is the socket (internal/realtimews). Both call the same
// store, so neither owns a rule the other lacks, and announcing happens below
// both (ADR-0010).
package trackerhttp

import (
	"context"
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Scopes resolves a project into its tracker. Declared here, in the consumer:
// this package never sees a workspace path or a room name, only a store
// already rooted where it belongs (ADR-0009).
type Scopes interface {
	Tracker(ctx context.Context, id projects.ID) (tracker.Store, error)
}

// Register mounts the tracker routes on mux, all under one project:
//
//	GET    …/tracker/schema                    the types
//	PUT    …/tracker/schema/{type}             put one type
//	DELETE …/tracker/schema/{type}             remove one type
//	GET    …/tracker/items                     find, filtered by query
//	POST   …/tracker/items                     create
//	GET    …/tracker/items/{item}              read one
//	PATCH  …/tracker/items/{item}?version=N    update
//	DELETE …/tracker/items/{item}?version=N    delete
//	GET    …/tracker/items/{item}/comments     list, oldest first
//	POST   …/tracker/items/{item}/comments     add
//
// Absolute patterns, because the project is a path segment in the middle:
// stripping a prefix to mount this would throw away the part it needs.
//
// There is no authentication, by design (ADR-0012).
func Register(mux *http.ServeMux, scopes Scopes) {
	const base = "/projects/{project}/tracker/"

	handle := func(pattern string, h func(http.ResponseWriter, *http.Request, tracker.Store)) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			store, err := scopes.Tracker(r.Context(), projects.ID(r.PathValue("project")))
			if err != nil {
				writeErr(w, err)
				return
			}
			h(w, r, store)
		})
	}

	handle("GET "+base+"schema", readSchema)
	handle("PUT "+base+"schema/{type}", putType)
	handle("DELETE "+base+"schema/{type}", deleteType)

	handle("GET "+base+"items", findItems)
	handle("POST "+base+"items", createItem)
	handle("GET "+base+"items/{item}", readItem)
	handle("PATCH "+base+"items/{item}", updateItem)
	handle("DELETE "+base+"items/{item}", deleteItem)

	handle("GET "+base+"items/{item}/comments", readComments)
	handle("POST "+base+"items/{item}/comments", addComment)
}

func itemID(r *http.Request) tracker.ItemID { return tracker.ItemID(r.PathValue("item")) }
