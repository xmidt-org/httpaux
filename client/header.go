package client

import (
	"net/http"

	"github.com/xmidt-org/httpaux"
)

// Header adds a middleware Constructor that uses a closure to modify
// each request header.  Both httpaux.Header.SetTo and httpaux.Header.AddTo
// can be used as this closure.
func Header(hf func(http.Header)) Constructor {
	return func(next httpaux.Client) httpaux.Client {
		return Func(func(request *http.Request) (*http.Response, error) {
			hf(request.Header)
			return next.Do(request)
		})
	}
}
