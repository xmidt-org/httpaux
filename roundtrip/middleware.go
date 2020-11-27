package roundtrip

import "net/http"

// Middleware is the interface implemented by components which can apply
// decoration to http.RoundTrippers.  Both Constructor and Chain
// implement this interface.
type Middleware interface {
	// Then applies decoration to the given round tripper.  See the note on
	// Constructor for information about the CloseIdleConnections method.
	Then(http.RoundTripper) http.RoundTripper
}

// Constructor applies clientside middleware to an http.RoundTripper.
//
// https://pkg.go.dev/net/http#Client.CloseIdleConnections
type Constructor func(http.RoundTripper) http.RoundTripper

// Then implements Middleware
func (c Constructor) Then(next http.RoundTripper) http.RoundTripper {
	return c(next)
}

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
