package httpmock

import (
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// RoundTripMethodName is the name of the http.RoundTripper.RoundTrip method.
// Used to start fluent expectation chains.
const RoundTripMethodName = "RoundTrip"

// RoundTripCall is syntactic sugar around a RoundTrip *mock.Call.
// This type provides some higher-level and typesafe expectation
// behavior.
//
// First, create a *RoundTripper.  Then, use a *RoundTripper's methods
// to create instances of this type to set expectations.
type RoundTripCall struct {
	*mock.Call

	// container is the RoundTripper that created this call.
	// used to access information about the enclosing test.
	container *RoundTripper

	// runFunc is the function that a client explicitly asked to be run.
	runFunc func(mock.Arguments)

	asserters []RequestAsserter
}

// newRoundTripCall properly initializes a RoundTripCall expectation.
func newRoundTripCall(container *RoundTripper, call *mock.Call) *RoundTripCall {
	rtc := &RoundTripCall{
		container: container,
		Call:      call,
	}

	rtc.Call.Run(rtc.run)
	return rtc
}

// run is the mock.Run implementation that executes
// any request assertions.  This call implementation always executes
// this function.
func (rtc *RoundTripCall) run(args mock.Arguments) {
	request, _ := args.Get(0).(*http.Request)
	rtc.container.applyAsserters(request, rtc.asserters)

	if rtc.runFunc != nil {
		rtc.runFunc(args)
	}
}

// Run establishes a run function for this mock.  This does not prevent
// any assertions from running.
func (rtc *RoundTripCall) Run(f func(mock.Arguments)) *RoundTripCall {
	rtc.runFunc = f
	return rtc
}

// Return establishes the return values for this RoundTrip invocation.
//
// If this method is not used, the Next round tripper set on the container
// will be used to generate the return.  If no Next has been set, this
// Call will fail the test when invoked.
func (rtc *RoundTripCall) Return(r *http.Response, err error) *RoundTripCall {
	rtc.Call = rtc.Call.Return(r, err)
	return rtc
}

// AssertRequest adds request assertions that are specific to this mocked Call.
// Multiple calls to this method are cumulative.
//
// Note that any global assertions on the mock that created this call will be
// executed in addition to these assertions.
func (rtc *RoundTripCall) AssertRequest(a ...RequestAsserter) *RoundTripCall {
	rtc.asserters = append(rtc.asserters, a...)
	return rtc
}

// RoundTripper is a mocked http.RoundTripper.  Instances should be
// created with NewRoundTripper or NewRoundTripperSuite.
//
// This type alters the Mock API slightly, since each instance is tied
// to a mock.TestingT instance.
type RoundTripper struct {
	mock.Mock

	// next is the delegate to which round trip calls are forwarded
	next http.RoundTripper

	t mock.TestingT

	assert    *assert.Assertions
	asserters []RequestAsserter
}

// NewRoundTripper returns a mock http.RoundTripper for the given test.
func NewRoundTripper(t mock.TestingT) *RoundTripper {
	rtc := new(RoundTripper)
	rtc.Test(t)
	return rtc
}

// NewRoundTripperSuite returns a mock http.RoundTripper for the given suite.
func NewRoundTripperSuite(s suite.TestingSuite) *RoundTripper {
	return NewRoundTripper(s.T())
}

// RoundTrip implements http.RoundTripper and is driven by the mock's expectations.
func (m *RoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	arguments := m.Called(request)
	if len(arguments) == 0 && m.next != nil {
		return m.next.RoundTrip(request)
	}

	var (
		first, _ = arguments.Get(0).(*http.Response)
		err, _   = arguments.Get(1).(error)
	)

	return first, err
}

// Next sets a delegate for this round tripper.  For any Calls that do not
// have an associated Return, this next instance will be used.
//
// If next is nil, http.DefaultTransport is used instead.
func (m *RoundTripper) Next(next http.RoundTripper) *RoundTripper {
	if next != nil {
		m.next = next
	} else {
		m.next = http.DefaultTransport
	}

	return m
}

// Test changes the test instance on this mock.
func (m *RoundTripper) Test(t mock.TestingT) {
	m.Mock.Test(t)
	m.t = t
	m.assert = assert.New(t)
}

// AssertRequest adds request assertions that apply to all mocked calls created
// via this instance.
//
// Setting global assertions simplifies cases where a mock is used for many
// requests that all must have some similarity.  For example, a test case may
// be testing something that always emits GET requests.
func (m *RoundTripper) AssertRequest(a ...RequestAsserter) *RoundTripper {
	m.asserters = append(m.asserters, a...)
	return m
}

// applyAsserters executes this mock's global assertions together with
// a slice of assertions defined on an individual Call expectation.
func (m *RoundTripper) applyAsserters(candidate *http.Request, local []RequestAsserter) {
	for _, a := range m.asserters {
		a.Assert(m.assert, candidate)
	}

	for _, a := range local {
		a.Assert(m.assert, candidate)
	}
}

// matchAny is the predicate used to unconditionally match any *http.Request.
func matchAny(*http.Request) bool { return true }

// OnAny is a convenience for starting a *mock.Call expectation which
// matches any HTTP request.
func (m *RoundTripper) OnAny() *RoundTripCall {
	return newRoundTripCall(
		m,
		m.On(RoundTripMethodName, mock.MatchedBy(matchAny)),
	)
}

// OnRequest starts a *mock.Call fluent chain that expects a call to
// RoundTrip with a request.  The expectation requires the exact request
// instance given.
//
// If middleware may submit a different request to this mock, use
// MatchAll or MatchAny instead.  For example, http.Request.WithContext
// creates a new request instance with the same state.
func (m *RoundTripper) OnRequest(request *http.Request) *RoundTripCall {
	return newRoundTripCall(
		m,
		m.On(RoundTripMethodName, mock.MatchedBy(
			func(candidate *http.Request) bool {
				return candidate == request
			},
		)),
	)
}

// OnMatchAll starts a *mock.Call fluent chain that expects a RoundTrip
// call with a request that matches all the given predicates.  If no
// predicates are supplied, the returned expectation will match any request.
func (m *RoundTripper) OnMatchAll(rms ...RequestMatcher) *RoundTripCall {
	return newRoundTripCall(
		m,
		m.On(RoundTripMethodName, mock.MatchedBy(
			func(candidate *http.Request) bool {
				for _, rm := range rms {
					if !rm.Match(candidate) {
						return false
					}
				}

				return true
			},
		)),
	)
}

// OnMatchAny starts a *mock.Call fluent chain expects a RoundTrip call
// with a request that matches any of the given predicates.  If no predicates
// are supplied, the returned expectation won't match any requests.
func (m *RoundTripper) OnMatchAny(rms ...RequestMatcher) *RoundTripCall {
	return newRoundTripCall(
		m,
		m.On(RoundTripMethodName, mock.MatchedBy(
			func(candidate *http.Request) bool {
				for _, rm := range rms {
					if rm.Match(candidate) {
						return true
					}
				}

				return false
			},
		)),
	)
}

// AssertExpectations uses the TestingT instance set at construction or with Test
// to assert all the calls have been executed.
func (m *RoundTripper) AssertExpectations() {
	m.Mock.AssertExpectations(m.t)
}

// CloseIdler is a mocked httpaux.CloseIdler that also mocks http.RoundTripper.
// Typical construction is:
//
//   func(t *testing.T) {
//     ci := &CloseIdler{
//       RoundTripper: NewRoundTripper(t),
//     }
//   }
type CloseIdler struct {
	*RoundTripper
}

// CloseIdleConnections implements httpaux.CloseIdler and is driven by this mock's expectations.
func (m *CloseIdler) CloseIdleConnections() {
	m.Called()
}

// OnCloseIdleConnections starts a fluent chain for defining a CloseIdleConnections expectation.
func (m *CloseIdler) OnCloseIdleConnections() *mock.Call {
	return m.On("CloseIdleConnections")
}
