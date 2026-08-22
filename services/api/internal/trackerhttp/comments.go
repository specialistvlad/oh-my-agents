package trackerhttp

import (
	"net/http"
	"strconv"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

type commentsBody struct {
	Comments []tracker.Comment `json:"comments"`
	Next     tracker.Cursor    `json:"next,omitempty"`
}

func readComments(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	page, err := s.Comments(r.Context(), itemID(r), pageOf(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	comments := page.Rows
	if comments == nil {
		comments = []tracker.Comment{}
	}
	writeJSON(w, http.StatusOK, commentsBody{Comments: comments, Next: page.Next})
}

// addComment posts to the item in the path, which is authoritative: a body
// naming another item would put the comment somewhere the caller did not ask.
func addComment(w http.ResponseWriter, r *http.Request, s tracker.Store) {
	var n tracker.NewComment
	if !decode(w, r, &n) {
		return
	}
	n.Item = itemID(r)
	posted, err := s.AddComment(r.Context(), n)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, posted)
}

// pageOf reads paging from the query. A bad limit is ignored rather than
// refused: the store applies its own default, and failing a read over a
// malformed page size helps nobody.
func pageOf(r *http.Request) tracker.PageRequest {
	page := tracker.PageRequest{Cursor: tracker.Cursor(r.URL.Query().Get("cursor"))}
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 {
		page.Limit = n
	}
	return page
}
