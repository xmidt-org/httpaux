package httpmock

import (
	"bytes"
	"io"
	"net/http"

	"github.com/stretchr/testify/assert"
)

// RequestMatcher supplies expectation matching for requests.  Implementations
// are used in mock.MatchedBy functions in order to select which expectation
// applies.
type RequestMatcher interface {
	// Match tests a request.
	Match(*http.Request) bool
}

// RequestMatcherFunc allows closures to be used directly as RequestMatchers
type RequestMatcherFunc func(*http.Request) bool

// Match satisifes the RequestMatcher interface
func (rmf RequestMatcherFunc) Match(r *http.Request) bool {
	return rmf(r)
}

// RequestAsserter executes assertions against requests.  Implementations are
// used in mock.Run functions to verify that a request is in the correct state.
// This is sometimes preferable to matching, which is done before an expectation
// is selected rather than after.
type RequestAsserter interface {
	// Assert asserts that a request is in the correct state.  This method
	// is used in mock.Run functions to verify a request after an expectation
	// has been matched.
	Assert(*assert.Assertions, *http.Request)
}

// RequestAsserterFunc allows closures to be used directly as RequestAsserters
type RequestAsserterFunc func(*assert.Assertions, *http.Request)

// Assert satisfies the RequestAsserter interface
func (raf RequestAsserterFunc) Assert(a *assert.Assertions, r *http.Request) {
	raf(a, r)
}

// RequestChecker is a combined strategy for both matching HTTP requests
// and asserting HTTP request state.
//
// Implementations must guarantee that any request for which Match returns
// true must also pass all assertions in Assert.  The reverse is also true:
// any request for which Match returns false must also fail Assert.
type RequestChecker interface {
	RequestMatcher
	RequestAsserter
}

// NopRequestChecker is a RequestChecker that does nothing.  This function
// is useful instead of nil.
type NopRequestChecker struct{}

// Match always returns true.
func (nrc NopRequestChecker) Match(*http.Request) bool { return true }

// Assert does nothing.
func (nrc NopRequestChecker) Assert(*assert.Assertions, *http.Request) {}

type methods []string

// Methods verifies that a request has one of several expected methods.
// An empty expected slice means that no request will match.
func Methods(expected ...string) RequestChecker {
	return append(
		make(methods, 0, len(expected)),
		expected...,
	)
}

func (m methods) Match(r *http.Request) bool {
	for _, v := range m {
		if r.Method == v {
			return true
		}
	}

	return false
}

func (m methods) Assert(assert *assert.Assertions, r *http.Request) {
	if assert.NotEmpty(m, "No methods defined") {
		if len(m) == 1 {
			// using Equal instead of Contains for (1) method makes test
			// failure output easier to read
			assert.Equal(
				m[0],
				r.Method,
				"The request method did not match",
			)
		} else {
			assert.Contains(
				m,
				r.Method,
				"The request method did not match",
			)
		}
	}
}

// Path verifies a request's URL.Path property.  This checker will also fail
// if the request's URL field is nil.
type Path string

func (p Path) Match(r *http.Request) bool {
	return r.URL != nil && r.URL.Path == string(p)
}

func (p Path) Assert(assert *assert.Assertions, r *http.Request) {
	if assert.NotNil(r.URL, "No URL set on the request") {
		assert.Equal(
			string(p),
			r.URL.Path,
			"The request URL.Path did not match",
		)
	}
}

type headerChecker struct {
	name     string
	expected []string
}

func (hc headerChecker) Match(r *http.Request) bool {
	actual := r.Header.Values(hc.name)
	if len(hc.expected) != len(actual) {
		return false
	} else if len(actual) == 0 {
		// both are empty, so all good
		return true
	}

	actualSet := make(map[string]bool, len(actual))
	for _, v := range actual {
		actualSet[v] = true
	}

	for _, e := range hc.expected {
		if !actualSet[e] {
			return false
		}
	}

	return true
}

func (hc headerChecker) Assert(assert *assert.Assertions, r *http.Request) {
	assert.ElementsMatch(
		hc.expected,
		r.Header.Values(hc.name),
		"The request header values did not match",
	)
}

// Header asserts that a given header has the expected values.  All header
// values must match the expected slice exactly.  If no expected values
// are passed, then the header must have no values.
func Header(name string, expected ...string) RequestChecker {
	return headerChecker{
		name:     http.CanonicalHeaderKey(name),
		expected: append([]string{}, expected...),
	}
}

// Body is a RequestAsserter asserts that the request body matches an expected value exactly.
// Passing an empty expected value is the same as asserting that there is
// no body.
//
// IMPORTANT: This asserter has to fully read the request body.  That means
// that any subsequent asserters won't have access to the request body.
type Body string

// Assert verifies that either:
//
// (1) The request body is nil AND this expected Body is the empty string
// (2) The request body is not nil AND it matches this expected Body exactly
func (b Body) Assert(assert *assert.Assertions, r *http.Request) {
	if r.Body == nil {
		// if the body is nil, the only possible match is an empty string
		assert.Empty(string(b), "The request body was nil, but text was expected")
		return
	}

	var actual bytes.Buffer
	_, err := io.Copy(&actual, r.Body)
	if assert.NoError(err, "Error reading request body") {
		assert.Equal(
			string(b),
			actual.String(),
			"The request body did not match the expected text",
		)
	}
}

// BodyJSON is a RequestAsserter that checks the request body against an
// expected JSON document.  Passing an empty expected value is the
// same as asserting that there is no body.
//
// IMPORTANT: This asserter has to fully read the request body.  That means
// that any subsequent asserters won't have access to the request body.
type BodyJSON string

// Assert verifies that either:
//
// (1) The request body is nil AND this expected JSON is an empty string
// (2) The request body is not nil AND it matches this expected JSON.  The
//     match is done via stretchr/testify/assert.JSONEq.
func (b BodyJSON) Assert(assert *assert.Assertions, r *http.Request) {
	if r.Body == nil {
		// if the body is nil, the only possible match is an empty string
		assert.Empty(string(b), "The request body was nil, but text was expected")
		return
	}

	var actual bytes.Buffer
	_, err := io.Copy(&actual, r.Body)
	if assert.NoError(err, "Error reading request body") {
		assert.JSONEq(
			string(b),
			actual.String(),
			"The request body did not match the expected JSON",
		)
	}
}
