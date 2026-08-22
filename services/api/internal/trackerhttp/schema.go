package trackerhttp

import (
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func readSchema(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	schema, err := s.Schema(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	if schema.Types == nil {
		schema.Types = []tracker.ItemType{}
	}
	writeJSON(w, http.StatusOK, schema)
}

// putType writes a type whole. The id in the path is authoritative, so a body
// naming a different one is refused rather than quietly saved somewhere the
// caller did not ask for.
func putType(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	var t tracker.ItemType
	if !decode(w, r, &t) {
		return
	}
	id := tracker.TypeID(r.PathValue("type"))
	if t.ID == "" {
		t.ID = id
	}
	if t.ID != id {
		writeJSON(w, http.StatusBadRequest, errBody{
			Error: "the body names type " + string(t.ID) + " but the path names " + string(id),
		})
		return
	}
	if err := s.PutItemType(r.Context(), t); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func deleteType(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	if err := s.DeleteItemType(r.Context(), tracker.TypeID(r.PathValue("type"))); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
