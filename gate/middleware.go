package httpaux

import "net/http"

// Server provides a simple, default middleware for controlling
// handlers based on a gate.  For additional control, particularly
// on what happens when the gate is closed, use ServerCustom.
func Server(g Interface) func(http.Handler) http.Handler {
	return ServerCustom{Gate: g}.Then
}

// ServerCustom defines a serverside middleware that controls access to handlers
// based upon a gate status.  This type provides additional options beyond what
// the Server function does.
type ServerCustom struct {
	// Closed is the optional handler to be invoked with the gate is closed.
	// If this field is not set, http.StatusServiceUnavailable is written
	// to the response.
	//
	// A convenient, configurable handler for this field is httpaux.ConstantHandler.
	Closed http.Handler

	// Gate is the required Status that indicates whether a gate allows traffic
	Gate Status
}

// Then decorates a handler so that it is controlled by the Gate field.  Next is required
// and cannot be nil, or a panic will result.
func (sc ServerCustom) Then(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case sc.Gate.IsOpen():
			next.ServeHTTP(response, request)

		case sc.Closed != nil:
			sc.Closed.ServeHTTP(response, request)

		default:
			response.WriteHeader(http.StatusServiceUnavailable)
		}
	})
}

// Client provides a simple, defaulted middleware for controlling outbound http
// traffic via a gate.  For more control over decoration, use a ClientCustom.
func Client(g Interface) func(http.RoundTripper) http.RoundTripper {
	return ClientCustom{Gate: g}.Then
}

// ClientCustom defines a clientside middleware that controls access to round trippers
// based upon a gate status.  This type provides additional options over and above
// what the Client constructor provides.
type ClientCustom struct {
	// Closed is the optional round tripper invoked when the gate is closed.  If this
	// field is unset, then a nil *http.Response and a *ClosedError are returned when
	// the gate status indicates closed.
	Closed http.RoundTripper

	// Gate is the required Status that indicates whether a gate allows traffic
	Gate Status
}

// Then decorates a round tripper so that it is controlled by the Gate field.
//
// The returned http.RoundTripper will always supply a CloseIdleConnections method.
// If next also supplies that method, it will be invoked whenever the decorator's method
// is invoked.  Otherwise, the decorator's CloseIdleConnections will do nothing.
//
// For consistency with other libraries, if next is nil then http.DefaultTransport
// is used as the decorated round tripper.
func (cc ClientCustom) Then(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return &roundTripper{
		next:   next,
		closed: cc.Closed,
		gate:   cc.Gate,
	}
}

type roundTripper struct {
	next   http.RoundTripper
	closed http.RoundTripper
	gate   Status
}

func (rt *roundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	switch {
	case rt.gate.IsOpen():
		return rt.next.RoundTrip(request)

	case rt.closed != nil:
		return rt.closed.RoundTrip(request)

	default:
		return nil, &ClosedError{Gate: rt.gate}
	}
}

func (rt *roundTripper) CloseIdleConnections() {
	type closeIdler interface {
		CloseIdleConnections()
	}

	if ci, ok := rt.next.(closeIdler); ok {
		ci.CloseIdleConnections()
	}
}
