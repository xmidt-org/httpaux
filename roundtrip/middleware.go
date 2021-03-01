package roundtrip

import (
	"net/http"
)

// Func is a function that that implements http.RoundTripper.
type Func func(*http.Request) (*http.Response, error)

// RoundTrip invokes this function and returns the results
func (f Func) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

var _ http.RoundTripper = Func(nil)

// Constructor applies clientside middleware to an http.RoundTripper.
//
// Care should be taken not to hide the CloseIdleConnections method of the
// given round tripper.  Otherwise, a containing http.Client's CloseIdleConnections
// method will be a noop.  The Decorator type in this package facilitates
// decoration of round trippers while preserving CloseIdleConnections behavior.
// Constructors executed as part of a Chain preserve this behavior automatically.
type Constructor func(http.RoundTripper) http.RoundTripper

// Chain is an immutable sequence of constructors.  This type is essentially
// a bundle of middleware for HTTP clients.
type Chain struct {
	c []Constructor
}

// NewChain creates a chain from a sequence of constructors.  The constructors
// are always applied in the order presented here.
func NewChain(c ...Constructor) Chain {
	return Chain{
		c: append([]Constructor{}, c...),
	}
}

// Append adds additional Constructors to this chain, and returns the new chain.
// This chain is not modified.  If more has zero length, this chain is returned.
func (c Chain) Append(more ...Constructor) Chain {
	if len(more) > 0 {
		return Chain{
			c: append(
				append([]Constructor{}, c.c...),
				more...,
			),
		}
	}

	return c
}

// Extend is like Append, except that the additional Constructors come from
// another chain
func (c Chain) Extend(more Chain) Chain {
	return c.Append(more.c...)
}

// Then applies the given sequence of middleware to the next http.RoundTripper.  In keeping
// with the de facto standard with net/http, if next is nil, then http.DefaultTransport
// is decorated.
//
// If next provides a CloseIdleConnections method, it is preserved and available
// in the returned http.RoundTripper.  Additionally, if any constructors decorate CloseIdleConnections,
// that decoration is preserved in the final product.  This enables the http.Client.CloseIdleConnections
// method to work properly.
//
// See: https://pkg.go.dev/net/http#Client.CloseIdleConnections
func (c Chain) Then(next http.RoundTripper) http.RoundTripper {
	if len(c.c) > 0 {
		if next == nil {
			next = http.DefaultTransport
		}

		// apply in reverse order, so that the order of
		// execution matches the order supplied to this chain
		for i := len(c.c) - 1; i >= 0; i-- {
			// preserve CloseIdleConnections at each level, so that if we have
			// a mix of constructors, some decorating that method and others not,
			// then we make sure CloseIdleConnections visits each decorator that
			// cares about that behavior
			next = PreserveCloseIdler(next, c.c[i](next))
		}
	}

	return next
}
