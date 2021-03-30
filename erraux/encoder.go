package erraux

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/xmidt-org/httpaux"
)

// StatusCoder can be implemented by errors to produce a custom status code.
type StatusCoder interface {
	StatusCode() int
}

// Headerer can be implemented by errors to produce custom headers.
type Headerer interface {
	Headers() http.Header
}

type errorEncoder func(context.Context, error, http.ResponseWriter) bool

// EncoderRule describes how to encode an error.  Rules are created with either Is or As.
type EncoderRule interface {
	// StatusCode sets the HTTP response code for this error.  This will override
	// any StatusCode() method supplied on the error.
	//
	// This method returns this rule for method chaining.
	StatusCode(int) EncoderRule

	// Headers adds header names and values according to the rules of httpaux.Header.AppendHeaders.
	// These will be added to any headers the error may define by implementing Headerer.
	//
	// This method returns this rule for method chaining.
	Headers(namesAndValues ...string) EncoderRule

	// Fields adds more JSON fields for this error.  These fields are added to the standard fields
	// and the fields an error returns through the ErrorFielder interface.  See the Fields.AddAll
	// method for how the variadic list of items is interpreted.
	//
	// This method returns this rule for method chaining.
	Fields(namesAndValues ...interface{}) EncoderRule

	// newErrorEncoder is the internal factory method that turns this rule
	// into something that can attempt to encode errors.
	newErrorEncoder() errorEncoder
}

type isEncoder struct {
	target     error
	statusCode int
	header     httpaux.Header
	fields     Fields
}

func (ie *isEncoder) newErrorEncoder() errorEncoder {
	body, _ := json.Marshal(ie.fields)
	return func(ctx context.Context, err error, rw http.ResponseWriter) bool {
		if !errors.Is(err, ie.target) {
			return false
		}

		ie.header.SetTo(rw.Header())
		rw.WriteHeader(ie.statusCode)
		rw.Write(body)
		return true
	}
}

func (ie *isEncoder) StatusCode(c int) EncoderRule {
	ie.statusCode = c
	ie.fields.SetCode(c)
	return ie
}

func (ie *isEncoder) Headers(v ...string) EncoderRule {
	ie.header = ie.header.AppendHeaders(v...)
	return ie
}

func (ie *isEncoder) Fields(namesAndValues ...interface{}) EncoderRule {
	ie.fields.Add(namesAndValues...)
	return ie
}

// Is creates an error EncoderRule that matches a target error.  errors.Is is used
// to determine if the returned rule matches.
//
// The supplied target is assumed to be immutable.  The status code, headers, and any
// extra fields are computed once and are an immutable part of the rule once it is
// added to a Matrix.
//
// See: https://pkg.go.dev/errors#Is
func Is(target error) EncoderRule {
	ie := &isEncoder{
		target:     target,
		statusCode: http.StatusInternalServerError,
	}

	var sc StatusCoder
	if errors.As(target, &sc) {
		ie.statusCode = sc.StatusCode()
	}

	var h Headerer
	if errors.As(target, &h) {
		ie.header = httpaux.NewHeader(h.Headers())
	}

	ie.fields = NewFields(ie.statusCode, target.Error())
	var ef ErrorFielder
	if errors.As(target, &ef) {
		ef.ErrorFields(ie.fields)
	}

	return ie
}

type asEncoder struct {
	target     reflect.Type
	statusCode int
	header     httpaux.Header
	fields     Fields
}

func (ae *asEncoder) newErrorEncoder() errorEncoder {
	return func(_ context.Context, err error, rw http.ResponseWriter) bool {
		targetPtr := reflect.New(ae.target)
		if !errors.As(err, targetPtr.Interface()) {
			return false
		}

		// in this case, the actual instance of the target type can vary.
		// so, we dynamically determine the things that Is can statically detect.

		ae.header.SetTo(rw.Header())
		target := targetPtr.Elem().Interface().(error)

		var h Headerer
		if errors.As(target, &h) {
			for k, v := range h.Headers() {
				k = http.CanonicalHeaderKey(k)
				rw.Header()[k] = v
			}
		}

		statusCode := ae.statusCode
		if statusCode <= 0 {
			var sc StatusCoder
			if errors.As(target, &sc) {
				statusCode = sc.StatusCode()
			}
		}

		if statusCode <= 0 {
			statusCode = http.StatusInternalServerError
		}

		rw.WriteHeader(statusCode)
		fields := NewFields(statusCode, target.Error())
		for k, v := range ae.fields {
			fields[k] = v
		}

		var ef ErrorFielder
		if errors.As(target, &ef) {
			ef.ErrorFields(fields)
		}

		body, _ := json.Marshal(fields)
		rw.Write(body)
		return true
	}
}

func (ae *asEncoder) StatusCode(c int) EncoderRule {
	ae.statusCode = c

	// NOTE: don't need to update the fields here, as the status code
	// will be dynamically determined.

	return ae
}

func (ae *asEncoder) Headers(v ...string) EncoderRule {
	ae.header = ae.header.AppendHeaders(v...)
	return ae
}

func (ae *asEncoder) Fields(namesAndValues ...interface{}) EncoderRule {
	ae.fields.Add(namesAndValues...)
	return ae
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// As creates an error EncoderRule which matches using errors.As.  The target must either
// be a pointer to an interface or a type which implements error.  Any other type will
// cause a panic.
//
// Because the error instance can vary, the rule created by this function dynamically
// determines the status code, headers, and fields using the optional interfaces
// defined in this package.
//
// Any status code set explicitly on the returned rule will override whatever an
// error defines at runtime.
//
// Any headers or fields set explicitly on the returned rule will be appended to
// whatever the error defines at runtime.
//
// See: https://pkg.go.dev/errors#As
func As(target interface{}) EncoderRule {
	if target == nil {
		// consistent with errors.As
		panic("erraux: target cannot be nil")
	}

	t := reflect.TypeOf(target)
	switch {
	case t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Interface:
		return &asEncoder{
			target: t.Elem(), // the interface type: errors.As((*MyInterface)(nil))
			fields: Fields{},
		}

	case t.Implements(errorType):
		return &asEncoder{
			target: t, // the exact type passed in: errors.As((*net.DNSError)(nil))
			fields: Fields{},
		}

	default:
		panic("erraux: target must either (1) be an interface pointer or (2) implement error")
	}
}

// Encoder defines a ruleset for writing golang errors to HTTP representations.
// The zero value is a valid matrix that will simply encode all errors with a default
// JSON representation.
//
// This type does not honor json.Marshaler.  Each error being allowed to marshal itself
// as needed leads to many different error representations with no guarantee of being
// consistent.  Rather, this type enforces a standard JSON representation and supports
// optional interfaces that custom errors may implement that can tailor parts of the
// HTTP response.
type Encoder struct {
	encoders []errorEncoder
}

// Add defines rules created with either Is or As.
func (e Encoder) Add(rules ...EncoderRule) Encoder {
	for _, r := range rules {
		e.encoders = append(e.encoders, r.newErrorEncoder())
	}

	return e
}

// Encode is a gokit-style error encoder.  Each rule in this matrix is tried in the order
// they were added via Add.  If no rule can handle the error, a default JSON representation
// is used.
//
// The minimal JSON error representation has two fields: code and cause.  The code is the HTTP
// status code, and will be the same as what was passed to WriteHeader.  The cause field is
// the value of Error().  For example:
//
//   {"code": 404, "cause": "resource not found"}
//   {"code": 500, "cause": "parsing error"}
//
// Beyond that, errors may implement StatusCoder, Headerer, and ErrorFielder
// to tailor the HTTP representation.  Any status code set on the rules will override any
// value the error supplies.  For example:
//
//   type MyError struct{}
//   func (e *MyError) StatusCode() int { return 504 }
//   func (e *MyError) Error() string { "my error" }
//
//   // this will override MyError.StatusCode
//   e := Encoder{}.Add(
//       As((*MyError)(nil)).StatusCode(500),
//   )
//
// By contrast, an headers or fields set on the rule will be appended to whatever the
// error defines at runtime.
//
// This method may be used with go-kit's transport/http package as an ErrorEncoder.
func (e Encoder) Encode(ctx context.Context, err error, rw http.ResponseWriter) {
	// always output JSON
	rw.Header().Set("Content-Type", "application/json")
	for _, rule := range e.encoders {
		if rule(ctx, err, rw) {
			return
		}
	}

	// fallback
	rw.WriteHeader(http.StatusInternalServerError)
	body, _ := json.Marshal(NewFields(
		http.StatusInternalServerError,
		err.Error(),
	))

	rw.Write(body)
}
