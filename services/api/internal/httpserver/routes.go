package httpserver

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
)

// newHandler builds the mux. Unexported at package scope so tests can
// exercise the routes without binding a port.
func newHandler(cfg Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET "+cfg.Prefix+"/health-check", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET "+cfg.Prefix+"/build-info.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprintf(w,
			"app_name: oh-my-agents-api\n"+
				"commit_hash: %s\n"+
				"build_time: %s\n"+
				"environment_name: %s\n",
			cfg.Commit, cfg.BuildTime, cfg.Environment)
	})

	for _, m := range cfg.Mounts {
		at := cfg.Prefix + m.Prefix
		mux.Handle(at, http.StripPrefix(strings.TrimSuffix(at, "/"), m.Handler))
	}

	// Profiling exposes runtime/heap detail and an unauthenticated CPU
	// profile is a DoS lever, so it stays opt-in. Mounted at the standard
	// root rather than under Prefix so `go tool pprof` works unchanged.
	if cfg.Profiling {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	return mux
}
