package server

import (
	"net/http"
)

// Header adds a middleware Constructor that uses a closure to modify
// each request header.  Both httpaux.Header.SetTo and httpaux.Header.AddTo
// can be used as this closure.
func Header(hf func(http.Header)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// apply header first, as decorated handlers will probably call WriteHeader
			// before returning
			hf(rw.Header())

			next.ServeHTTP(rw, r)
		})
	}
}
