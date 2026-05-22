// Package server provides the `region serve` local preview HTTP server.
// It wraps http.FileServer with two pieces of middleware:
//
//   - CORS (Access-Control-Allow-Origin: *) so the playground can fetch the
//     local API from a different origin if needed.
//   - Cache-Control for *.json so browsers behave like they would behind a CDN.
//
// This is a dev/preview tool, not a production server.
package server

import (
	"log"
	"net/http"
	"path"
	"strings"
)

// Run starts a blocking HTTP server serving files from dir on addr.
func Run(addr, dir string) error {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(dir))
	mux.Handle("/", middleware(fs))

	log.Printf("region serve: %s → %s", addr, dir)
	log.Printf("open http://localhost%s/", addr)
	return http.ListenAndServe(addr, mux)
}

// middleware adds CORS + cache headers and request logging.
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if strings.HasSuffix(path.Ext(r.URL.Path), "json") {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
