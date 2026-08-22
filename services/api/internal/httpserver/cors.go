package httpserver

import (
	"net/http"
	"slices"
	"strconv"
	"time"
)

// preflightMaxAge is how long a browser may cache a preflight answer. Long
// enough that a busy page is not re-asking constantly, short enough that a
// change to the allowed origins takes effect the same day.
const preflightMaxAge = 10 * time.Minute

// allowedMethods and allowedHeaders describe the whole API surface. Both are
// closed lists rather than reflections of the request, because echoing back
// whatever was asked for is how a permissive CORS policy gets written by
// accident.
var (
	allowedMethods = "GET, POST, PUT, DELETE, OPTIONS"
	allowedHeaders = "Content-Type"
)

// cors makes the API usable from the web app, which is served from a
// different port and is therefore a different origin.
//
// The origin list is the same one the WebSocket handler enforces. A socket is
// not subject to CORS and an HTTP request is, so without this the socket
// connects and every fetch beside it fails — which is exactly what the
// browser reports as a bare "Load failed".
//
// Credentials are never allowed: nothing here uses cookies, and permitting
// them would make a wildcard origin genuinely dangerous rather than merely
// broad.
func cors(allowed []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// The answer depends on the origin, so caches must not share it.
		w.Header().Add("Vary", "Origin")

		if origin != "" && originAllowed(allowed, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(int(preflightMaxAge.Seconds())))
		}
		// A preflight is answered here and never reaches a route: the mux
		// knows nothing about OPTIONS and would call it a 405, which a
		// browser reports as a failed fetch rather than a refused origin.
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// originAllowed reports whether this origin may read responses. "*" allows
// any, which is for local development and tests.
func originAllowed(allowed []string, origin string) bool {
	return slices.Contains(allowed, "*") || slices.Contains(allowed, origin)
}
