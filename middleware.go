package httpaux

import "net/http"

// ServerMiddleware represents a bundle of decorators for HTTP handlers.
// justinas/alice.Chain implements this interface.
type ServerMiddleware interface {
	Then(http.Handler) http.Handler
}

// ClientMiddleware represents a bundle of decorators for HTTP round trippers.
// The roundtrip package provides implementations of this interface.
type ClientMiddleware interface {
	ThenRoundTrip(http.RoundTripper) http.RoundTripper
}
