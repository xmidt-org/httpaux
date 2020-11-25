package httpaux

import "net/http"

// StateChange indicates what to do to a gate
type StateChange int

const (
	// StateNoChange indicates that no control operation should be performed
	StateNoChange StateChange = iota

	// StateOpen indicates that the gate should be opened
	StateOpen

	// StateClose indicates that the gate should be closed
	StateClose
)

// ControlHandler is an http.Handler that allows HTTP requests to open or close a gate
type ControlHandler struct {
	// StateChange is the required strategy for determining how to change a gate
	// given an HTTP request.  This closure can examine any aspect of the request, e.g.
	// URL parameters, URI, etc, to determine what state change should occur.
	//
	// This closure can return StateNoChange to indicate that the handler should do nothing
	// to a gate.  This may be true due to a security permissions check, as one example.
	StateChange func(*http.Request) StateChange

	// Control is the required gate control instance used to open and close the gate
	Control Control
}

// ServeHTTP invokes the StateChange closure and takes the appropriate action
func (ch ControlHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch ch.StateChange(request) {
	case StateOpen:
		ch.Control.Open()

	case StateClose:
		ch.Control.Close()
	}
}
