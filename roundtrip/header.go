// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package roundtrip

import (
	"net/http"
)

// Header adds a middleware Constructor that uses a closure to modify
// each request header.  Both httpaux.Header.SetTo and httpaux.Header.AddTo
// can be used as this closure.
func Header(hf func(http.Header)) Constructor {
	return func(next http.RoundTripper) http.RoundTripper {
		return Func(func(request *http.Request) (*http.Response, error) {
			hf(request.Header)
			return next.RoundTrip(request)
		})
	}
}
