package erraux

import "net/http"

const (
	codeFieldName  = "code"
	causeFieldName = "cause"
)

// ErrorFielder can be implemented by errors to produce custom fields
// in the rendered JSON.
type ErrorFielder interface {
	// ErrorFields can add any desired fields that flesh out the error.
	ErrorFields(Fields)
}

// Fields holds JSON fields for an error.
type Fields map[string]interface{}

// NewFields constructs the minimal JSON fields for an error.
// If code is less than 100 (the lowest value HTTP status code),
// then http.StatusInternalServerError is used instead.
func NewFields(code int, cause string) Fields {
	if code < 100 {
		code = http.StatusInternalServerError
	}

	return Fields{
		codeFieldName:  code,
		causeFieldName: cause,
	}
}

// SetCode updates the code field.
func (f Fields) SetCode(code int) {
	f[codeFieldName] = code
}

// HasCause tests if this Fields has a cause.
func (f Fields) HasCause() bool {
	_, ok := f[causeFieldName]
	return ok
}

// SetCause updates the cause field.
func (f Fields) SetCause(cause string) {
	f[causeFieldName] = cause
}

// Clone returns a distinct, shallow copy of this Fields instance.
func (f Fields) Clone() Fields {
	c := make(Fields, len(f))
	for k, v := range f {
		c[k] = v
	}

	return c
}

// Merge merges the fields from the given Fields into this instance.
func (f Fields) Merge(more Fields) {
	for k, v := range more {
		f[k] = v
	}
}

// Add adds a variadic set of names and values to this fields.
//
// Each even-numbered item in this method's variadic arguments must be a string, or
// this method will panic.  Each odd-numbered item is paired as the value of the preceding
// name.  If there are an odd number of items, the last item must be a string and it
// is interpreted as having an empty value.
func (f Fields) Add(namesAndValues ...interface{}) {
	for i, j := 0, 1; i < len(namesAndValues); i, j = i+2, j+2 {
		name := namesAndValues[i].(string)
		var value interface{}
		if j < len(namesAndValues) {
			value = namesAndValues[j]
		}

		f[name] = value
	}
}
