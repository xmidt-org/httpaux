// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package httpaux

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ConstantHandler is an http.Handler that writes a statically defined HTTP response.
// Very useful for testing and for default behavior.
type ConstantHandler struct {
	// StatusCode is the response code to pass to http.ResponseWriter.WriteHeader.
	// If this value is less than 100 (which also includes being unset), then no
	// response code is written which will trigger the default of http.StatusOK.
	StatusCode int

	// Header is the set of headers added to every response
	Header Header

	// ContentType is the MIME type of the Body.  This will override anything written by
	// the Headers closure.  This field has no effect if Body is not also set.  If Body is
	// set and this field is unset, then no Content-Type header is written by this handler.
	// A Content-Type may still be written by other infrastructure or by the Headers closure, however.
	ContentType string

	// Body is the optional HTTP entity body.  If unset, nothing is written for the response body.
	// A Content-Length header will be explicitly set if this field is set.
	Body []byte
}

// ServeHTTP returns the constant information in the response.
func (ch ConstantHandler) ServeHTTP(response http.ResponseWriter, _ *http.Request) {
	ch.Header.SetTo(response.Header())
	l := len(ch.Body)
	if l > 0 {
		response.Header().Set("Content-Length", strconv.Itoa(l))
		if len(ch.ContentType) > 0 {
			response.Header().Set("Content-Type", ch.ContentType)
		}
	}

	if ch.StatusCode >= 100 {
		response.WriteHeader(ch.StatusCode)
	}

	if l > 0 {
		response.Write(ch.Body)
	}
}

// ConstantJSON is a convenience for constructing a ConstantHandler that returns
// a static JSON message.  The encoding/json package is used to marshal v.
func ConstantJSON(v interface{}) (ch ConstantHandler, err error) {
	ch.Body, err = json.Marshal(v)
	if err == nil {
		ch.ContentType = "application/json; charset=utf-8"
	}

	return
}
