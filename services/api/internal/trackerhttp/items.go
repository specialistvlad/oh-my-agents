package trackerhttp

import (
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

type itemsBody struct {
	Items []tracker.Item `json:"items"`
	Next  tracker.Cursor `json:"next,omitempty"`
}

func findItems(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	q, err := parseQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody{Error: err.Error()})
		return
	}
	page, err := s.FindItems(r.Context(), q)
	if err != nil {
		writeErr(w, err)
		return
	}
	// Always an array, so a client never has to handle null.
	items := page.Rows
	if items == nil {
		items = []tracker.Item{}
	}
	writeJSON(w, http.StatusOK, itemsBody{Items: items, Next: page.Next})
}

func createItem(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	var n tracker.NewItem
	if !decode(w, r, &n) {
		return
	}
	created, err := s.CreateItem(r.Context(), n)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func readItem(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	item, err := s.Item(r.Context(), itemID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func updateItem(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	expected, err := version(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody{Error: err.Error()})
		return
	}
	var p tracker.Patch
	if !decode(w, r, &p) {
		return
	}
	updated, err := s.UpdateItem(r.Context(), itemID(r), expected, p)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func deleteItem(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	expected, err := version(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody{Error: err.Error()})
		return
	}
	// Who is deleting travels in the body, because a delete has no other
	// place to put it and the feed is worth attributing accurately.
	var by struct {
		Author tracker.ActorRef `json:"author"`
	}
	if r.ContentLength > 0 && !decode(w, r, &by) {
		return
	}
	if err := s.DeleteItem(r.Context(), itemID(r), expected, by.Author); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
