package erraux

import "net/http"

const (
	codeFieldName  = "code"
	causeFieldName = "cause"
)

// ErrorFielder can be implemented by errors to produce custom fields
// in the rendered JSON.
type ErrorFielder interface {
	// ErrorFields can return any desired fields that flesh out the error.
	// The returned slice is in the same format as Fields.Add.
	ErrorFields() []interface{}
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

// Code returns the status code for this set of fields.
// This method returns 0 if there is no code or if the code
// is not an int.
func (f Fields) Code() int {
	c, _ := f[codeFieldName].(int)
	return c
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

// Cause returns the cause for this set of fields.
// This method returns the empty string if there is no cause
// or if the cause is not a string.
func (f Fields) Cause() string {
	c, _ := f[causeFieldName].(string)
	return c
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
// is interpreted as having an nil value.
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

// Append does the reverse of Add.  This method flattens
// a Fields into a sequence of {name1, value1, name2, value2, ...} values.
func (f Fields) Append(nav []interface{}) []interface{} {
	if cap(nav) < len(nav)+len(f) {
		grow := make([]interface{}, 0, len(nav)+len(f))
		grow = append(grow, nav...)
		nav = grow
	}

	for k, v := range f {
		nav = append(nav, k, v)
	}

	return nav
}
