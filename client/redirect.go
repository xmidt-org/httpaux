// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"net/http"
)

// CheckRedirect is the type expected by http.Client.CheckRedirect.
//
// Closures of this type can also be chained together via the
// NewCheckRedirects function.
type CheckRedirect func(*http.Request, []*http.Request) error

// CopyHeadersOnRedirect copies the headers from the most recent
// request into the next request.  If no names are supplied, this
// function returns nil so that the default behavior will take over.
func CopyHeadersOnRedirect(names ...string) CheckRedirect {
	if len(names) == 0 {
		return nil
	}

	names = append([]string{}, names...)
	for i, n := range names {
		names[i] = http.CanonicalHeaderKey(n)
	}

	return func(request *http.Request, via []*http.Request) error {
		previous := via[len(via)-1] // the most recent request
		for _, n := range names {
			// direct map access is faster, since we've already
			// canonicalized everything
			if values := previous.Header[n]; len(values) > 0 {
				request.Header[n] = values
			}
		}

		return nil
	}
}

// MaxRedirects returns a CheckRedirect that returns an error if
// a maximum number of redirects has been reached.  If the max
// value is 0 or negative, then no redirects are allowed.
func MaxRedirects(max int) CheckRedirect {
	if max < 0 {
		max = 0
	}

	// create the error once and reuse it
	// this error text mimics the one used in net/http
	err := fmt.Errorf("stopped after %d redirects", max)
	return func(_ *http.Request, via []*http.Request) error {
		if len(via) >= max {
			return err
		}

		return nil
	}
}

// NewCheckRedirects produces a CheckRedirect that is the logical AND
// of the given strategies.  All the checks must pass, or the returned
// function halts early and returns the error from the failing check.
//
// Since a nil http.Request.CheckRedirect indicates that the internal
// default will be used, this function returns nil if no checks are
// supplied.  Additionally, any nil checks are skipped.  If all checks
// are nil, this function also returns nil.
func NewCheckRedirects(checks ...CheckRedirect) CheckRedirect {
	if len(checks) == 0 {
		return nil
	}

	// check nils before allocating a copy
	for _, c := range checks {
		if c == nil {
			return nil
		}
	}

	// optimization:  if there's only (1) check, just use that
	if len(checks) == 1 {
		return checks[0]
	}

	checks = append([]CheckRedirect{}, checks...)
	return func(request *http.Request, via []*http.Request) (err error) {
		for i := 0; err == nil && i < len(checks); i++ {
			err = checks[i](request, via)
		}

		return
	}
}
