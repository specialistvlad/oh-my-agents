// Package projectshttp exposes the project lifecycle over HTTP.
//
// The other edge is the socket (internal/realtimews). Both call the same
// store, so neither owns a rule the other lacks, and announcing happens below
// both (ADR-0010).
package projectshttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
)

// maxBody caps a request. A project record is a name and a path.
const maxBody = 64 << 10

// Store is what serving projects needs and nothing more.
type Store interface {
	Create(ctx context.Context, n projects.New) (projects.Project, error)
	Get(ctx context.Context, id projects.ID) (projects.Project, error)
	List(ctx context.Context) ([]projects.Project, error)
	Rename(ctx context.Context, id projects.ID, name string) (projects.Project, error)
	Repoint(ctx context.Context, id projects.ID, root string) (projects.Project, error)
	Remove(ctx context.Context, id projects.ID) error
}

// change is the body of an edit. Both fields are optional; whichever is
// present is applied.
type change struct {
	Name *string `json:"name,omitempty"`
	Root *string `json:"root,omitempty"`
}

// Register mounts the project routes on mux:
//
//	GET    /projects/      list
//	POST   /projects/      create
//	GET    /projects/{id}  read one
//	PATCH  /projects/{id}  rename and/or re-point
//	DELETE /projects/{id}  remove, which deletes the root directory
//
// Absolute patterns, so that per-project resources can register alongside
// these without either package owning the other's URLs.
//
// There is no authentication, by design (ADR-0012). That matters most at
// DELETE, which removes the project's root directory and everything in it.
func Register(mux *http.ServeMux, s Store) {
	mux.HandleFunc("GET /projects/{$}", func(w http.ResponseWriter, r *http.Request) {
		all, err := s.List(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, listBody{Projects: nonNil(all)})
	})
	mux.HandleFunc("POST /projects/{$}", func(w http.ResponseWriter, r *http.Request) {
		var in projects.New
		if !decode(w, r, &in) {
			return
		}
		created, err := s.Create(r.Context(), in)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	})
	mux.HandleFunc("GET /projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := s.Get(r.Context(), id(r))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	})
	mux.HandleFunc("PATCH /projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		patch(w, r, s)
	})
	mux.HandleFunc("DELETE /projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Remove(r.Context(), id(r)); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// patch applies a rename, a re-point, or both. Both are separate calls on the
// store because they mean different things — one is display text, the other
// claims a directory — and a body doing both does them in that order.
func patch(w http.ResponseWriter, r *http.Request, s Store) {
	var in change
	if !decode(w, r, &in) {
		return
	}
	if in.Name == nil && in.Root == nil {
		writeJSON(w, http.StatusBadRequest, errBody{Error: "nothing to change"})
		return
	}
	p, err := s.Get(r.Context(), id(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	if in.Name != nil {
		if p, err = s.Rename(r.Context(), p.ID, *in.Name); err != nil {
			writeErr(w, err)
			return
		}
	}
	if in.Root != nil {
		if p, err = s.Repoint(r.Context(), p.ID, *in.Root); err != nil {
			writeErr(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, p)
}

func id(r *http.Request) projects.ID { return projects.ID(r.PathValue("id")) }

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, errBody{Error: "body too large"})
		return false
	}
	if err := json.Unmarshal(body, into); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody{Error: "malformed body"})
		return false
	}
	return true
}

// writeErr maps a store failure onto a status. Anything unrecognized is a 500
// with no detail, because an unexpected error is as likely to describe the
// host as the request.
func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, projects.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errBody{Error: err.Error()})
	case errors.Is(err, projects.ErrInvalidID),
		errors.Is(err, projects.ErrInvalidName),
		errors.Is(err, projects.ErrInvalidRoot):
		writeJSON(w, http.StatusBadRequest, errBody{Error: err.Error()})
	case errors.Is(err, projects.ErrNotAProjectRoot):
		// The refusal that keeps a mistake from becoming a recursive
		// delete. A conflict, not a bad request: the request was fine and
		// the directory is not what the record says it is.
		writeJSON(w, http.StatusConflict, errBody{Error: err.Error()})
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusRequestTimeout, errBody{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errBody{Error: "internal error"})
	}
}

type listBody struct {
	Projects []projects.Project `json:"projects"`
}

type errBody struct {
	Error string `json:"error"`
}

func nonNil(all []projects.Project) []projects.Project {
	if all == nil {
		return []projects.Project{}
	}
	return all
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
