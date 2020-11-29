package gate

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
	// URL parameters, request URI, etc, to determine what state change should occur.
	//
	// This closure can return StateNoChange to indicate that the handler should do nothing
	// to a gate.  For example, the request may fail a security permissions check.
	StateChange func(*http.Request) StateChange

	// Gate is the required gate control instance used to open and close the gate
	Gate Control
}

// ServeHTTP invokes the StateChange closure and takes the appropriate action
func (ch ControlHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch ch.StateChange(request) {
	case StateOpen:
		ch.Gate.Open()

	case StateClose:
		ch.Gate.Close()
	}
}
