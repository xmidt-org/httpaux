package httpmock

import (
	"net/http"

	"github.com/stretchr/testify/assert"
)

// RequestAssertion is used during the RoundTripper's Run function to
// execute assertions against the request.  It's often better to do
// this instead of using a match, since the test fail message is clearer.
type RequestAssertion func(*assert.Assertions, *http.Request)

// NopRequestAssertion is a RequestAssertion that does nothing.
//
// This function is useful instead of nil.
func NopRequestAssertion(*assert.Assertions, *http.Request) {}

// Methods asserts that a request's method is any of an expected number of methods.
// Methods must match exactly, i.e. must be properly uppercased.
//
// If no methods are supplied, NopRequestAssertion is returned.
func Methods(expected ...string) RequestAssertion {
	switch len(expected) {
	case 0:
		return NopRequestAssertion

	case 1:
		// using Equal instead of Contains for (1) method makes test
		// failure output easier to read
		return func(assert *assert.Assertions, candidate *http.Request) {
			assert.Equal(
				expected[0],
				candidate.Method,
				"The request method did not match",
			)
		}

	default:
		return func(assert *assert.Assertions, candidate *http.Request) {
			assert.Contains(
				expected,
				candidate.Method,
				"The request method did not match",
			)
		}
	}
}

// Path asserts that a request's URL.Path match an expected value.
// The returned assertion will also fail if a request has no URL field set.
func Path(expected string) RequestAssertion {
	return func(assert *assert.Assertions, candidate *http.Request) {
		if assert.NotNil(candidate.URL, "No URL set on the request") {
			assert.Equal(
				expected,
				candidate.URL.Path,
				"The request URL.Path did not match",
			)
		}
	}
}

// Header asserts that a given header has the expected values.  All header
// values must match the expected slice exactly.
//
// If no expected values are passed, NopRequestAssertion is returned.
func Header(name string, expected ...string) RequestAssertion {
	if len(expected) == 0 {
		return NopRequestAssertion
	}

	return func(assert *assert.Assertions, candidate *http.Request) {
		assert.ElementsMatchf(
			expected,
			candidate.Header.Values(name),
			"The request header [%s] did not match",
			name,
		)
	}
}
