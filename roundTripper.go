package httpaux

import "net/http"

// RoundTripperFunc is a function that that implements http.RoundTripper
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip invokes this function and returns the results
func (rtf RoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return rtf(request)
}

var _ http.RoundTripper = RoundTripperFunc(nil)

type closeIdler interface {
	CloseIdleConnections()
}

// roundTripper is the internal http.RoundTripper implementation used to
// preserve CloseIdleConnections in chains of middleware.
type roundTripper struct {
	http.RoundTripper
	closeIdler
}

// CloseIdleConnections is a helper for preserving the CloseIdleConnections behavior of an
// http.RoundTripper when a middleware doesn't wish to decorate that method.
//
// If decorated provides a CloseIdleConnections method, it is returned as is.
//
// If next provides a CloseIdleConnections method, and decorated does not, then
// an http.RoundTripper is returned that delegates RoundTrip calls to the decorated
// instance and CloseIdleConnections calls to the next instance.
//
// If neither next nor decorated provides a CloseIdleConnections method, then this
// method does nothing and simply returns decorated.
func CloseIdleConnections(next, decorated http.RoundTripper) http.RoundTripper {
	if _, ok := decorated.(closeIdler); ok {
		return decorated
	} else if ci, ok := next.(closeIdler); ok {
		return roundTripper{
			RoundTripper: decorated,
			closeIdler:   ci,
		}
	}

	return decorated
}

// ClientMiddleware is the interface implemented by components which can apply
// decoration to http.RoundTrippers.  Both RoundTripperConstructor and RoundTripperChain
// implement this interface.
type ClientMiddleware interface {
	// Then applies decoration to the given round tripper.  See the note on
	// RoundTripperConstructor for information about the CloseIdleConnections method.
	Then(http.RoundTripper) http.RoundTripper
}

// RoundTripperConstructor applies clientside middleware to an http.RoundTripper.
//
// IMPORTANT: If a constructor returns an http.RoundTripper that does not provide
// a CloseIdleConnections method, then the http.Client.CloseIdleConnections will not be
// able to close idle connections when using that http.RoundTripper.  The RoundTripperChain
// type handles this case by exposing the CloseIdleConnections method of the original
// http.RoundTripper in the case where constructors do not decorate CloseIdleConnections.
// The CloseIdleConnections function of this package allows individual constructors to
// preserve this behavior.
//
// For example:
//
//   // the http.RoundTripper returned by this constructor hides any CloseIdleConnections
//   // implementation in next
//   func Simple(next http.RoundTripper) http.RoundTripper {
//     return http.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
//       // etc
//     })
//   }
//
//   // the returned http.RoundTripper preserves any CloseIdleConnections behavior in
//   // next, despite this constructor not being concerned with that method
//   func PreservesCloseIdleConnections(next http.RoundTripper) http.RoundTripper {
//     return CloseIdleConnections(
//       next,
//       http.RoundTripperFunc(func(*http.Request) (*http.Respnse, error) {
//         // etc
//       }),
//     )
//   }
//
// https://pkg.go.dev/net/http#Client.CloseIdleConnections
type RoundTripperConstructor func(http.RoundTripper) http.RoundTripper

// Then implements ClientMiddleware
func (rtc RoundTripperConstructor) Then(next http.RoundTripper) http.RoundTripper {
	return rtc(next)
}

// RoundTripperChain is an immutable sequence of constructors.  This type is essentially
// a bundle of middleware for HTTP clients.
type RoundTripperChain struct {
	c []RoundTripperConstructor
}

// NewRoundTripperChain creates a chain from a sequence of constructors.  The constructors
// are always applied in the order presented here.
func NewRoundTripperChain(c ...RoundTripperConstructor) RoundTripperChain {
	return RoundTripperChain{
		c: append([]RoundTripperConstructor{}, c...),
	}
}

// Append adds additional RoundTripperConstructors to this chain, and returns the new chain.
// This chain is not modified.  If more has zero length, this chain is returned.
func (lc RoundTripperChain) Append(more ...RoundTripperConstructor) RoundTripperChain {
	if len(more) > 0 {
		return RoundTripperChain{
			c: append(
				append([]RoundTripperConstructor{}, lc.c...),
				more...,
			),
		}
	}

	return lc
}

// Extend is like Append, except that the additional RoundTripperConstructors come from
// another chain
func (lc RoundTripperChain) Extend(more RoundTripperChain) RoundTripperChain {
	return lc.Append(more.c...)
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
func (lc RoundTripperChain) Then(next http.RoundTripper) http.RoundTripper {
	if len(lc.c) > 0 {
		if next == nil {
			next = http.DefaultTransport
		}

		// apply in reverse order, so that the order of
		// execution matches the order supplied to this chain
		for i := len(lc.c) - 1; i >= 0; i-- {
			// preserve CloseIdleConnections at each level, so that if we have
			// a mix of constructors, some decorating that method and others not,
			// then we make sure CloseIdleConnections visits each decorator that
			// cares about that behavior
			next = CloseIdleConnections(next, lc.c[i](next))
		}
	}

	return next
}
