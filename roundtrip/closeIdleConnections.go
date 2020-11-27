package roundtrip

import "net/http"

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
	} else if rt, ok := next.(roundTripper); ok {
		// optimization: carry over the closeIdler and drop the unnecessary decoration
		return roundTripper{
			RoundTripper: decorated,
			closeIdler:   rt.closeIdler,
		}
	} else if ci, ok := next.(closeIdler); ok {
		return roundTripper{
			RoundTripper: decorated,
			closeIdler:   ci, // preserve next's CloseIdleConnections behavior
		}
	}

	return decorated
}
