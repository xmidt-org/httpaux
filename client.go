// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package httpaux

import (
	"io"
	"net/http"
)

// Client is the canonical interface implemented by *http.Client
type Client interface {
	Do(*http.Request) (*http.Response, error)
}

var _ Client = (*http.Client)(nil)

// Cleanup is a utility function for ensuring that a client response's
// Body is drained and closed.  This function does not set the Body to nil.
//
// If either the response or the response.Body field is nil, this function
// does nothing.
func Cleanup(r *http.Response) {
	if r != nil && r.Body != nil {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}
}
