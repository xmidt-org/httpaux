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
	// StatusCode returns the HTTP response code for an error.  Typically, this
	// will be a 4xx or 5xx code.  However, this package allows any valid HTTP
	// code to be used.
	StatusCode() int
}

// statusCodeFor is a helper that determines the status code for an error.
// If err or anything in its chain implements StatusCoder, that value is used.
// Otherwise, this function returns http.StatusInternalServerError.
func statusCodeFor(err error) (statusCode int) {
	var sc StatusCoder
	if errors.As(err, &sc) {
		statusCode = sc.StatusCode()
	}

	if statusCode < 100 {
		// this case includes if a StatusCoder returned a bad value
		statusCode = http.StatusInternalServerError
	}

	return
}

// Headerer can be implemented by errors to produce custom headers.
type Headerer interface {
	Headers() http.Header
}

// headersFor is a helper that determines the error-specific headers.
// If err or anything in its change implements Headerer, those headers are used.
// Otherwise, this function returns an empty http.Header.
func headersFor(err error) (headers http.Header) {
	var h Headerer
	if errors.As(err, &h) {
		headers = h.Headers()
	}

	return
}

// Causer can be implemented by errors to produce a cause, which corresponds
// to the JSON field of the same name.  An error may choose to do this because
// the Error() method might be targeted at logs or other plain text output, while
// Cause() is specifically for a JSON representation.
type Causer interface {
	Cause() string
}

// causeFor is a helper that determines the cause of an error.  If err or
// anything in its chain implements Causer, that value is used.  Otherwise,
// Error() is used.
func causeFor(err error) (cause string) {
	var c Causer
	if errors.As(err, &c) {
		cause = c.Cause()
	} else {
		cause = err.Error()
	}

	return
}

// errorContext represents an encoding context for an error.
type errorContext struct {
	ctx context.Context
	err error
	rw  http.ResponseWriter
}

// errorEncoder represents an actual rule implementation that can
// render an error as an HTTP response.  If the given error did not
// match anything this rule could render, it will return false.
// If this rule rendered the error, it returns true.
type errorEncoder func(errorContext) bool

// EncoderRule describes how to encode an error.  Rules are created with either Is or As.
type EncoderRule interface {
	// StatusCode sets the HTTP response code for this error.  This will override
	// any StatusCode() method supplied on the error.
	//
	// This method returns this rule for method chaining.
	StatusCode(int) EncoderRule

	// Cause sets the cause field for this error.  This will override any Cause() method
	// supplied on the error.
	//
	// This method returns this rule for method chaining.
	Cause(string) EncoderRule

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
	newErrorEncoder(disableBody bool) errorEncoder
}

type isEncoder struct {
	target     error
	statusCode int
	header     httpaux.Header
	fields     Fields
}

// encodeNoBody conditionally encodes the error's status code and
// headers, returning a boolean indicating if it handled the error.
func (ie *isEncoder) encodeNoBody(ec errorContext) bool {
	if !errors.Is(ec.err, ie.target) {
		return false
	}

	ie.header.SetTo(ec.rw.Header())
	ec.rw.WriteHeader(ie.statusCode)
	return true
}

func (ie *isEncoder) newErrorEncoder(disableBody bool) errorEncoder {
	if disableBody {
		return ie.encodeNoBody
	}

	body, _ := json.Marshal(ie.fields)
	return func(ec errorContext) bool {
		handled := ie.encodeNoBody(ec)
		if handled {
			ec.rw.Write(body)
		}

		return handled
	}
}

func (ie *isEncoder) StatusCode(c int) EncoderRule {
	ie.statusCode = c
	ie.fields.SetCode(c)
	return ie
}

func (ie *isEncoder) Cause(v string) EncoderRule {
	ie.fields.SetCause(v)
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
// added to an Encoder.
//
// See: https://pkg.go.dev/errors#Is
func Is(target error) EncoderRule {
	ie := &isEncoder{
		target: target,
		header: httpaux.NewHeader(headersFor(target)),
		fields: make(Fields, 2), // at least enough for code and cause
	}

	ie.StatusCode(statusCodeFor(target))
	ie.Cause(causeFor(target))
	ie.fields.Add(fieldsFor(target)...)

	return ie
}

type asEncoder struct {
	target     reflect.Type
	statusCode int
	header     httpaux.Header
	fields     Fields
}

func (ae *asEncoder) encodeStatusAndHeaders(ec errorContext) (target error, statusCode int, handled bool) {
	targetPtr := reflect.New(ae.target)
	handled = errors.As(ec.err, targetPtr.Interface())
	if handled {
		// in this case, the actual instance of the target type can vary.
		// so, we dynamically determine the things that Is can statically detect.
		ae.header.SetTo(ec.rw.Header())
		target = targetPtr.Elem().Interface().(error)

		for k, v := range headersFor(target) {
			ec.rw.Header()[http.CanonicalHeaderKey(k)] = v
		}

		statusCode = ae.statusCode
		if statusCode < 100 {
			statusCode = statusCodeFor(target)
		}

		ec.rw.WriteHeader(statusCode)
	}

	return
}

func (ae *asEncoder) encodeNoBody(ec errorContext) bool {
	_, _, handled := ae.encodeStatusAndHeaders(ec)
	return handled
}

func (ae *asEncoder) newErrorEncoder(disableBody bool) errorEncoder {
	if disableBody {
		return ae.encodeNoBody
	}

	return func(ec errorContext) bool {
		target, statusCode, handled := ae.encodeStatusAndHeaders(ec)
		if handled {
			fields := ae.fields.Clone()
			fields.SetCode(statusCode)
			if !fields.HasCause() {
				fields.SetCause(causeFor(target))
			}

			fields.Add(fieldsFor(target)...)
			body, _ := json.Marshal(fields)
			ec.rw.Write(body)
		}

		return handled
	}
}

func (ae *asEncoder) StatusCode(c int) EncoderRule {
	ae.statusCode = c

	// NOTE: don't need to update the fields here, as the status code
	// will be dynamically determined.

	return ae
}

func (ae *asEncoder) Cause(v string) EncoderRule {
	ae.fields.SetCause(v)
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

	var targetType reflect.Type
	t := reflect.TypeOf(target)
	switch {
	case t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Interface:
		targetType = t.Elem() // the interface type: errors.As((*MyInterface)(nil))

	case t.Implements(errorType):
		targetType = t // the exact type passed in: errors.As((*net.DNSError)(nil))

	default:
		// consistent with errors.As
		panic("erraux: target must either (1) be an interface pointer or (2) implement error")
	}

	return &asEncoder{
		target: targetType,
		fields: Fields{},
	}
}

// Encoder defines a ruleset for writing golang errors to HTTP representations.
// The zero value is a valid encoder that will simply encode all errors with a default
// JSON representation.
//
// This type does not honor json.Marshaler.  Each error being allowed to marshal itself
// as needed leads to many different error representations with no guarantee of being
// consistent.  Rather, this type enforces a standard JSON representation and supports
// optional interfaces that custom errors may implement that can tailor parts of the
// HTTP response.
type Encoder struct {
	encoders    []errorEncoder
	disableBody bool
}

// Body controls whether HTTP response bodies are rendered for any rules added
// afterward.  By default, an Encoder will render a JSON payload for each error.
//
// Invoking this method only affects subsequent calls up to the next invocation:
//
//   Encoder{}.
//     Body(false).
//       // these errors will be rendered without a body
//       Is(ErrSomething).
//       As((*SomeError)(nil).
//     Body(true).
//       // this error will be rendered with a body
//       As((*MyError)(nil)) // this will be rendered with a body
func (e Encoder) Body(v bool) Encoder {
	e.disableBody = !v
	return e
}

// Add defines rules created with either Is or As.
func (e Encoder) Add(rules ...EncoderRule) Encoder {
	for _, r := range rules {
		e.encoders = append(e.encoders, r.newErrorEncoder(e.disableBody))
	}

	return e
}

// Encode is a gokit-style error encoder.  Each rule in this encoder is tried in the order
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
//   e := erraux.Encoder{}.Add(
//       erraux.As((*MyError)(nil)).StatusCode(500),
//   )
//
// By contrast, an headers or fields set on the rule will be appended to whatever the
// error defines at runtime.
//
// This method may be used with go-kit's transport/http package as an ErrorEncoder.
func (e Encoder) Encode(ctx context.Context, err error, rw http.ResponseWriter) {
	// always output JSON
	rw.Header().Set("Content-Type", "application/json")

	ec := errorContext{
		ctx: ctx,
		err: err,
		rw:  rw,
	}

	for _, rule := range e.encoders {
		if rule(ec) {
			return
		}
	}

	// fallback
	// still honor the basic interfaces in this package, even though
	// there are no rules
	for k, v := range headersFor(err) {
		ec.rw.Header()[http.CanonicalHeaderKey(k)] = v
	}

	statusCode := statusCodeFor(err)
	rw.WriteHeader(statusCode)
	if !e.disableBody {
		fields := NewFields(
			statusCode,
			causeFor(err),
		)

		fields.Add(fieldsFor(err)...)
		body, _ := json.Marshal(fields)
		rw.Write(body)
	}
}
