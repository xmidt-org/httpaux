package erraux

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Error is a convenient carrier of error information that exposes
// HTTP response information.  This type implements several interfaces
// in popular packages like go-kit.
//
// This type is provided for code that is HTTP-aware, e.g. the presentation tier.
// For more general code, prefer defining an Encoder with custom rules to define
// the HTTP representation of errors.
type Error struct {
	// Err is the cause of this error.  This field is required.
	//
	// Typically, this field is set to the service-layer error or other error
	// that occurred below the presentation layer.
	Err error

	// Message is the optional message to associate with this error.
	Message string

	// Code is the response code to use for this error.  If unset, http.StatusInternalServerError
	// is used instead.
	Code int

	// Header is the optional set of HTTP headers to associate with this error.
	Header http.Header

	// Fields is the optional set of extra fields associated with this error.
	Fields Fields
}

var _ StatusCoder = (*Error)(nil)
var _ Causer = (*Error)(nil)
var _ Headerer = (*Error)(nil)
var _ ErrorFielder = (*Error)(nil)

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

// Cause returns just the error's Error() result.  The usage of this
// method is intended to be JSON or other non-plaintext representations.
func (e *Error) Cause() string {
	return e.Err.Error()
}

// Headers returns the optional headers to associate with this error's response
func (e *Error) Headers() http.Header {
	return e.Header
}

// ErrorFields fulfills the ErrorFielder interface and allows this error to
// supply additional fields that describe the error.
func (e *Error) ErrorFields() []interface{} {
	nav := make([]interface{}, 0, len(e.Fields)+1)
	nav = e.Fields.Append(nav)
	if len(e.Message) > 0 {
		nav = append(nav, "message", e.Message)
	}

	return nav
}

// MarshalJSON allows this Error to be marshaled directly as JSON.
// The JSON representation is consistent with Encoder.  However, when
// used with an Encoder, this method is not used.
func (e *Error) MarshalJSON() ([]byte, error) {
	f := NewFields(e.Code, e.Cause())
	f.Merge(e.Fields)
	if len(e.Message) > 0 {
		f["message"] = e.Message
	}

	return json.Marshal(f)
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
