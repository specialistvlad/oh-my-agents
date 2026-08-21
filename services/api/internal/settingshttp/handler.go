// Package settingshttp exposes a settings store over HTTP.
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

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// maxBody caps a stored document. Settings are small by nature, and an
// unbounded reader writing straight to disk is not something to leave open.
const maxBody = 1 << 20

// Store is what serving settings needs and nothing more.
type Store interface {
	Get(ctx context.Context, key settings.Key) (settings.Document, error)
	Set(ctx context.Context, key settings.Key, doc settings.Document) error
	Delete(ctx context.Context, key settings.Key) error
	Keys(ctx context.Context) ([]settings.Key, error)
}

// New returns a handler serving one store:
//
//	GET    /          list keys
//	GET    /{key...}  read one document
//	PUT    /{key...}  write one document
//	DELETE /{key...}  remove one document
//
// It is mounted under a prefix by the caller, so the patterns here are
// relative and the handler does not care where it lives.
//
// There is no authentication. Nothing in this service has any yet, and until
// that changes this must not be exposed beyond a trusted network.
func New(s Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		keys, err := s.Keys(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, keysBody{Keys: nonNil(keys)})
	})
	mux.HandleFunc("GET /{key...}", func(w http.ResponseWriter, r *http.Request) {
		doc, err := s.Get(r.Context(), key(r))
		if err != nil {
			writeErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(doc)
	})
	mux.HandleFunc("PUT /{key...}", func(w http.ResponseWriter, r *http.Request) {
		put(w, r, s)
	})
	mux.HandleFunc("DELETE /{key...}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Delete(r.Context(), key(r)); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func put(w http.ResponseWriter, r *http.Request, s Store) {
	doc, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, errBody{Error: err.Error()})
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
	case errors.Is(err, settings.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errBody{Error: err.Error()})
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
