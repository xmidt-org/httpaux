package httpmock

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

// RoundTripper is a mocked http.RoundTripper
type RoundTripper struct {
	mock.Mock
}

// RoundTrip implements http.RoundTripper and is driven by the mock's expectations
func (m *RoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	var (
		arguments = m.Called(request)
		first, _  = arguments.Get(0).(*http.Response)
		err, _    = arguments.Get(1).(error)
	)

	return first, err
}

// RoundTripCall is syntactic sugar around a RoundTrip *mock.Call
type RoundTripCall struct {
	*mock.Call
}

// Return establishes the return values for this RoundTrip invocation
func (rtc RoundTripCall) Return(r *http.Response, err error) RoundTripCall {
	rtc.Call.Return(r, err)
	return rtc
}

// OnRoundTrip starts a *mock.Call fluent chain to define an expectation
//
// The expected parameter is an interface{} allow not only a *http.Request
// but also a MatchedBy predicate.
func (m *RoundTripper) OnRoundTrip(expected interface{}) RoundTripCall {
	return RoundTripCall{
		Call: m.On("RoundTrip", expected),
	}
}

// CloseIdler is a mocked httpaux.CloseIdler that also mocks http.RoundTripper
type CloseIdler struct {
	RoundTripper
}

// CloseIdleConnections implements httpaux.CloseIdler and is driven by this mock's expectations
func (m *CloseIdler) CloseIdleConnections() {
	m.Called()
}

// OnCloseIdleConnections starts a fluent chain for defining a CloseIdleConnections expectation
func (m *CloseIdler) OnCloseIdleConnections() *mock.Call {
	return m.On("CloseIdleConnections")
}
