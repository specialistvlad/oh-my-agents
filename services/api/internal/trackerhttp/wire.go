package trackerhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// maxBody caps a request. An item is a title, a body and some fields.
const maxBody = 1 << 20

type errBody struct {
	Error string `json:"error"`
}

// writeErr maps a store failure onto a status.
//
// The mapping is the contract both edges share, so a client learns the same
// thing whichever it used. Anything unrecognized is a 500 with no detail,
// because an unexpected error is as likely to describe the host as the
// request.
func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, StatusOf(err), errBody{Error: message(err)})
}

// StatusOf is the shared failure mapping, exported because the socket edge
// answers with the same codes (ADR-0010).
func StatusOf(err error) int {
	switch {
	case errors.Is(err, tracker.ErrNotFound), errors.Is(err, projects.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, tracker.ErrVersionConflict),
		errors.Is(err, tracker.ErrUnresolvedDescendants),
		errors.Is(err, tracker.ErrResolvedParent),
		errors.Is(err, tracker.ErrHasChildren),
		errors.Is(err, tracker.ErrCycle):
		// A conflict, not a bad request: the call was well formed and the
		// tracker's current state is what refuses it.
		return http.StatusConflict
	case errors.Is(err, tracker.ErrTransitionNotAllowed),
		errors.Is(err, tracker.ErrUnknownType),
		errors.Is(err, tracker.ErrUnknownField),
		errors.Is(err, tracker.ErrUnknownStatus),
		errors.Is(err, tracker.ErrUnknownOption),
		errors.Is(err, tracker.ErrFieldRequired),
		errors.Is(err, tracker.ErrKindMismatch),
		errors.Is(err, tracker.ErrReservedKey),
		errors.Is(err, tracker.ErrInvalidSchema),
		errors.Is(err, tracker.ErrInvalidCursor),
		errors.Is(err, projects.ErrInvalidID):
		return http.StatusBadRequest
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return http.StatusRequestTimeout
	default:
		return http.StatusInternalServerError
	}
}

// message hides the detail of a failure we did not anticipate.
func message(err error) string {
	if StatusOf(err) == http.StatusInternalServerError {
		return "internal error"
	}
	return err.Error()
}

// version reads the compare-and-swap version a write must state.
//
// A query parameter rather than If-Match: it is legible in a log line and in
// a curl, and a version is a counter rather than an opaque validator. Absent
// is refused rather than defaulted, because a write that does not say what it
// expects is the overwrite this exists to prevent.
func version(r *http.Request) (tracker.Version, error) {
	raw := r.URL.Query().Get("version")
	if raw == "" {
		return 0, errors.New("this write needs ?version=N, the version it expects to replace")
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0, errors.New("version must be a non-negative whole number")
	}
	return tracker.Version(n), nil
}

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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
