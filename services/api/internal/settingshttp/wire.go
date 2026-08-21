package settingshttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// keysBody is the listing response.
type keysBody struct {
	Keys []settings.Key `json:"keys"`
}

// errBody is what every failure returns, so a client parses one shape.
type errBody struct {
	Error string `json:"error"`
}

// errTooLarge is a body over the cap, as opposed to one that could not be
// read at all. They are different failures and deserve different statuses.
var errTooLarge = errors.New("body too large")

// readBody reads a request body up to maxBody. The ResponseWriter is handed
// to MaxBytesReader so it can close a connection that keeps sending after the
// limit, which is the whole point of the cap.
func readBody(w http.ResponseWriter, r *http.Request) (settings.Document, error) {
	doc, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err == nil {
		return doc, nil
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return nil, fmt.Errorf("%w: %d bytes maximum", errTooLarge, maxBody)
	}
	return nil, fmt.Errorf("unreadable body: %w", err)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
