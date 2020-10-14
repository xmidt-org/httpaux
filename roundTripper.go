package httpbuddy

import "net/http"

// RoundTripperFunc is a function that that implements http.RoundTripper
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip invokes this function and returns the results
func (rtf RoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return rtf(request)
}

var _ http.RoundTripper = RoundTripperFunc(nil)
