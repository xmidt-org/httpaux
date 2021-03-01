package client

import (
	"net/http"

	"github.com/xmidt-org/httpaux"
)

// Func is an HTTP client function type
type Func func(*http.Request) (*http.Response, error)

// Do fulfills the httpaux.Client interface and permits this function
// to be used like an HTTP client.
func (f Func) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

var _ httpaux.Client = Func(nil)

// Constructor applies clientside middleware to an HTTP client, as implemented
// by httpaux.Client.
type Constructor func(httpaux.Client) httpaux.Client

// Chain is an immutable sequence of constructors.  This type is essentially
// a bundle of middleware for HTTP clients.
type Chain struct {
	c []Constructor
}

// NewChain creates a chain from a sequence of constructors.  The constructors
// are always applied in the order presented here.
func NewChain(ctors ...Constructor) (c Chain) {
	if len(ctors) > 0 {
		c.c = make([]Constructor, len(ctors))
		copy(c.c, ctors)
	}

	return
}

// Append adds additional Constructors to this chain, and returns the new chain.
// This chain is not modified.  If more has zero length, this chain is returned.
func (c Chain) Append(more ...Constructor) (nc Chain) {
	if len(more) > 0 {
		nc.c = make([]Constructor, 0, len(c.c)+len(more))
		nc.c = append(nc.c, c.c...)
		nc.c = append(nc.c, more...)
	} else {
		nc = c
	}

	return
}

// Extend is like Append, except that the additional Constructors come from
// another chain
func (c Chain) Extend(more Chain) Chain {
	return c.Append(more.c...)
}

// Then applies the given sequence of middleware to the next httpaux.Client.
func (c Chain) Then(next httpaux.Client) httpaux.Client {
	if next == nil {
		next = http.DefaultClient
	}

	if len(c.c) > 0 {
		// apply in reverse order, so that the order of
		// execution matches the order supplied to this chain
		for i := len(c.c) - 1; i >= 0; i-- {
			next = c.c[i](next)
		}
	}

	return next
}

// ThenFunc makes it easier to use a client transactor Func as the HTTP client
func (c Chain) ThenFunc(next Func) httpaux.Client {
	if next == nil {
		return c.Then(http.DefaultClient) // avoid a "nil" interface
	}

	return c.Then(next)
}
