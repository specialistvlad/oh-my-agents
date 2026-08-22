// Package settingshttp exposes a project's settings over HTTP.
//
// The interfaces below are declared here, in the consumer, and hold only the
// methods this handler calls (ADR-0002). Nothing in this package names a
// storage technology, so it serves an [settings.FS] and an
// [settings.Memory] identically — which is what the tests do.
package settingshttp

import (
	"context"
	"errors"
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// maxBody caps a stored document. Settings are small by nature, and an
// unbounded reader writing straight to disk is not something to leave open.
const maxBody = 1 << 20

// Scopes resolves a project into its settings store. Declared here, in the
// consumer: this package never sees a workspace path or a room name, only a
// store that is already rooted where it belongs (ADR-0009).
type Scopes interface {
	Settings(ctx context.Context, id projects.ID) (settings.Store, error)
}

// Store is what serving settings needs and nothing more.
//
// Announcing a write is not this package's business: it wraps whatever store
// it is handed, and a store that announces (internal/settingsbus) satisfies
// this interface unchanged. That is what keeps two write edges from each
// needing their own copy of the announcement.
type Store interface {
	Get(ctx context.Context, key settings.Key) (settings.Document, error)
	Set(ctx context.Context, key settings.Key, doc settings.Document) error
	Delete(ctx context.Context, key settings.Key) error
	Keys(ctx context.Context) ([]settings.Key, error)
}

// Register mounts the settings routes on mux:
//
//	GET    /projects/{project}/settings/          list keys
//	GET    /projects/{project}/settings/{key...}  read one document
//	PUT    /projects/{project}/settings/{key...}  write one document
//	DELETE /projects/{project}/settings/{key...}  remove one document
//
// Absolute patterns rather than a mounted subtree, because the project is a
// path segment in the middle: stripping a prefix to mount this would throw
// away the very part it needs.
//
// There is no authentication. Anything reachable here can read and write any
// project's settings, so this must not be exposed beyond a trusted network.
func Register(mux *http.ServeMux, scopes Scopes) {
	const base = "/projects/{project}/settings/"

	mux.HandleFunc("GET "+base+"{$}", func(w http.ResponseWriter, r *http.Request) {
		s, ok := storeFor(w, r, scopes)
		if !ok {
			return
		}
		keys, err := s.Keys(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, keysBody{Keys: nonNil(keys)})
	})
	mux.HandleFunc("GET "+base+"{key...}", func(w http.ResponseWriter, r *http.Request) {
		s, ok := storeFor(w, r, scopes)
		if !ok {
			return
		}
		doc, err := s.Get(r.Context(), key(r))
		if err != nil {
			writeErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(doc)
	})
	mux.HandleFunc("PUT "+base+"{key...}", func(w http.ResponseWriter, r *http.Request) {
		s, ok := storeFor(w, r, scopes)
		if !ok {
			return
		}
		put(w, r, s)
	})
	mux.HandleFunc("DELETE "+base+"{key...}", func(w http.ResponseWriter, r *http.Request) {
		s, ok := storeFor(w, r, scopes)
		if !ok {
			return
		}
		if err := s.Delete(r.Context(), key(r)); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// storeFor resolves the project in the path, answering the request itself if
// it cannot.
func storeFor(w http.ResponseWriter, r *http.Request, scopes Scopes) (Store, bool) {
	s, err := scopes.Settings(r.Context(), projects.ID(r.PathValue("project")))
	if err != nil {
		writeErr(w, err)
		return nil, false
	}
	return s, true
}

func put(w http.ResponseWriter, r *http.Request, s Store) {
	doc, err := readBody(w, r)
	if errors.Is(err, errTooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errBody{Error: err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody{Error: err.Error()})
		return
	}
	if err := s.Set(r.Context(), key(r), doc); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// key reads the wildcard segment. It is passed through unvalidated: the
// store rejects what it does not like, and duplicating that here would be
// the second implementation ADR-0005 warns about.
func key(r *http.Request) settings.Key {
	return settings.Key(r.PathValue("key"))
}

// writeErr maps a store failure onto a status. Anything unrecognized is a
// 500 with no detail, because an unexpected error is as likely to describe
// the host as the request.
func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, settings.ErrNotFound), errors.Is(err, projects.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errBody{Error: err.Error()})
	case errors.Is(err, projects.ErrInvalidID):
		writeJSON(w, http.StatusBadRequest, errBody{Error: err.Error()})
	case errors.Is(err, settings.ErrInvalidKey), errors.Is(err, settings.ErrInvalidDocument):
		writeJSON(w, http.StatusBadRequest, errBody{Error: err.Error()})
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusRequestTimeout, errBody{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errBody{Error: "internal error"})
	}
}

func nonNil(keys []settings.Key) []settings.Key {
	if keys == nil {
		return []settings.Key{}
	}
	return keys
}
