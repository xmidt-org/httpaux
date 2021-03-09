package httpaux

import (
	"bytes"
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
	o.WriteRune('{')
	if len(e.Message) > 0 {
		o.WriteString(`"message": "`)
		o.WriteString(e.Message)
		o.WriteString(`", `)
	}

	o.WriteString(`"cause": "`)
	o.WriteString(e.Err.Error())
	o.WriteString(`"}`)

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
