package httpaux

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Error is a convenient carrier of error information that exposes
// HTTP response information.  This type implements several interfaces
// in popular packages like go-kit.
//
// The primary use case for this type is wrapping errors so they can be
// easily rendered as HTTP responses.
type Error struct {
	// Err is the cause of this error.  This field is required.
	//
	// Typically, this field is set to the service-layer error or other error
	// that occurred below the presentation layer.
	Err error

	// Message is the optional message to associate with this error
	Message string

	// Code is the response code to use for this error.  If unset, http.StatusInternalServerError
	// is used instead.
	Code int

	// Header is the optional set of HTTP headers to associate with this error
	Header http.Header
}

// Unwrap produces the cause of this error
func (e *Error) Unwrap() error {
	return e.Err
}

// Error fulfills the error interface.  Message is included in this text
// if it is supplied.
func (e *Error) Error() string {
	if len(e.Message) > 0 {
		return fmt.Sprintf("%s: %s", e.Message, e.Err)
	}

	return e.Err.Error()
}

// StatusCode returns the Code field, or http.StatusInternalServerError if that field
// is less than 100.
func (e *Error) StatusCode() int {
	if e.Code < 100 {
		return http.StatusInternalServerError
	}

	return e.Code
}

// Headers returns the optional headers to associate with this error's response
func (e *Error) Headers() http.Header {
	return e.Header
}

// MarshalJSON produces a JSON representation of this error.  The Err field
// is marshaled as "cause".  If the Message field is set, it is marshaled as "message".
func (e *Error) MarshalJSON() ([]byte, error) {
	var o bytes.Buffer
	if len(e.Message) > 0 {
		fmt.Fprintf(&o, `{"code": %d, "message": "%s", "cause": "%s"}`, e.StatusCode(), e.Message, e.Err.Error())
	} else {
		fmt.Fprintf(&o, `{"code": %d, "cause": "%s"}`, e.StatusCode(), e.Err.Error())
	}

	return o.Bytes(), nil
}

// IsTemporary tests if the given error is marked as a temporary error.
// This method returns true if the given error or what the error wraps
// exposes a Temporary() bool method that returns true.
//
// This function uses errors.As to traverse the error wrappers.
//
// See: https://pkg.go.dev/net/#Error
func IsTemporary(err error) bool {
	type temporary interface {
		Temporary() bool
	}

	var te temporary
	if errors.As(err, &te) {
		return te.Temporary()
	}

	return false
}

// EncodeError is a gokit-style error coder for server code that consistently
// writes JSON for all errors.  The builtin gokit error encoder will write text/plain
// for anything it doesn't recognize, giving rise to inconsistent messages when
// some errors implement json.Marshaler and others do not.
//
// This function also honors embedded errors via the errors package.  If any error
// in the chain provides the specialized methods such as StatusCode, that error is used
// for that portion of the HTTP response.
func EncodeError(_ context.Context, err error, rw http.ResponseWriter) {
	type headerer interface {
		Headers() http.Header
	}

	var h headerer
	if errors.As(err, &h) {
		headers := h.Headers()
		for name := range headers {
			name = http.CanonicalHeaderKey(name)
			rw.Header()[name] = append(rw.Header()[name], headers[name]...)
		}
	}

	// always write JSON
	rw.Header().Set("Content-Type", "application/json")

	type statusCoder interface {
		StatusCode() int
	}

	code := http.StatusInternalServerError
	var sc statusCoder
	if errors.As(err, &sc) {
		code = sc.StatusCode()
	}

	rw.WriteHeader(code)

	var m json.Marshaler
	if errors.As(err, &m) {
		body, marshalErr := m.MarshalJSON()
		if marshalErr == nil {
			rw.Write(body)
			return
		}
	}

	// fallback to a simple JSON message
	simple, _ := json.Marshal(struct {
		Code  int    `json:"code"`
		Cause string `json:"cause"`
	}{
		Code:  code,
		Cause: err.Error(),
	})

	rw.Write(simple)
}
