package httpaux

import "net/http"

// Client is the canonical interface implemented by *http.Client
type Client interface {
	Do(*http.Request) (*http.Response, error)
}

var _ Client = (*http.Client)(nil)
