package settingshttp

import (
	"encoding/json"
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

// readBody reads a request body up to maxBody.
func readBody(r *http.Request) (settings.Document, error) {
	doc, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("body exceeds %d bytes", maxBody)
	}
	return doc, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
