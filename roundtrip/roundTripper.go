// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package roundtrip

import "net/http"

// CloseIdler is the strategy for closing idle connections.  This package
// makes this behavior explicit so that middleware will not hide this behavior.
type CloseIdler interface {
	CloseIdleConnections()
}

// CloseIdlerFunc is a function that implements CloseIdler.  Useful when the
// CloseIdleConnections method needs to be decorated.
type CloseIdlerFunc func()

func (cif CloseIdlerFunc) CloseIdleConnections() {
	cif()
}

// CloseIdleConnections invokes the CloseIdleConnections method of a round tripper
// if that object exposes that method.  Otherwise, this method does nothing.
//
// This function simplifies decoration code for CloseIdler.
func CloseIdleConnections(rt http.RoundTripper) {
	if ci, ok := rt.(CloseIdler); ok {
		ci.CloseIdleConnections()
	}
}

// Decorator is a convenience for decorating an http.RoundTripper in a manner that
// preserves CloseIdleConnections behavior.  This type is used automatically by
// the CloseIdleConnections function, but can be used for other custom decoration.
type Decorator struct {
	// RoundTripper is the object that receives RoundTrip calls.  Note that the Func
	// type in this package can be used here.
	http.RoundTripper

	// CloseIdler is the object that receives CloseIdleConnections calls.  When decorating
	// a round tripper, this field guarantees that an http.Client can close idle connections.
	CloseIdler
}

var _ http.RoundTripper = Decorator{}
var _ CloseIdler = Decorator{}

// PreserveCloseIdler is a helper for preserving the CloseIdleConnections behavior of an
// http.RoundTripper when a middleware doesn't wish to decorate that method.
//
// If decorator provides a CloseIdleConnections method, it is returned as is.
//
// If next provides a CloseIdleConnections method, and decorator does not, then
// an http.RoundTripper is returned that delegates RoundTrip calls to the decorator
// instance and CloseIdleConnections calls to the next instance.
//
// If neither next nor decorated provides a CloseIdleConnections method, then this
// method does nothing and simply returns decorator.
func PreserveCloseIdler(next, decorator http.RoundTripper) http.RoundTripper {
	if _, ok := decorator.(CloseIdler); ok {
		return decorator
	} else if d, ok := next.(Decorator); ok {
		// optimization: carry over the closeIdler and drop the unnecessary decoration
		return Decorator{
			RoundTripper: decorator,
			CloseIdler:   d.CloseIdler,
		}
	} else if ci, ok := next.(CloseIdler); ok {
		return Decorator{
			RoundTripper: decorator,
			CloseIdler:   ci, // preserve next's CloseIdleConnections behavior
		}
	}

	return decorator
}
