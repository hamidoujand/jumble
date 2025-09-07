// Package debug provides handler for debugging the application.
package debug

import (
	"expvar"
	"net/http"
	"net/http/pprof"
)

// Register returns a mux with debug endpoints register on it.
func Register() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/vars/", expvar.Handler())

	return mux
}
