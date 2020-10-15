package httpaux

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
)

const (
	gateOpen uint32 = iota
	gateClosed
)

var (
	gateOpenText   = "open"
	gateClosedText = "closed"
)

// GateClosedError is returned by decorated http.RoundTripper instances
// to indicate that the gate disallowed the client request.
type GateClosedError struct {
	// Gate is the gate instance that was closed at the time the attempt
	// to make a request was made.  Note that after this error returns, the
	// gate may be open.
	Gate *Gate
}

// Error satisfies the error interface
func (gce *GateClosedError) Error() string {
	return fmt.Sprintf("Gate [%s] closed", gce.Gate.Name())
}

// GateHook is a tuple of callbacks for Gate state.
//
// At least (1) callback is required, or an attempt to register the hook
// will be a nop.
type GateHook struct {
	// OnOpen is the callback for when a gate is open.  This callback
	// is passed Gate.Name().
	//
	// OnOpen callbacks are executed under a mutex so that state changes
	// are seen atomically.  Implementation should return in a timely fashion.
	OnOpen func(string)

	// OnClosed is the callback for when a gate is closed.  This callback
	// is passed Gate.Name().
	//
	// OnClosed callbacks are executed under a mutex so that state changes
	// are seen atomically.  Implementation should return in a timely fashion.
	OnClosed func(string)
}

// Gate is an atomic boolean that controls access to http.Handlers and http.RoundTrippers.
// All methods of this type are safe for concurrent access.
//
// A Gate can be observed via the Append method and supplying callbacks for state.  These
// callbacks are useful for integrating logging, metrics, health checks, etc.
type Gate struct {
	name     string
	onClosed http.Handler

	stateLock         sync.Mutex
	value             uint32
	onOpenCallbacks   []func(string)
	onClosedCallbacks []func(string)
}

// NewGate creates an unnamed Gate instance that returns http.StatusServiceUnavailable
// anytime the gate is closed.
func NewGate() *Gate {
	return NewGateCustom("", nil)
}

// NewGateCustom allows more control over the Gate creation.  It allows a name
// and a custom http.Handler that is invoked when the Gate is closed.
//
// If onClosed is nil, http.StatusServiceUnavailable is returned anytime the gate is closed.
func NewGateCustom(name string, onClosed http.Handler) *Gate {
	if onClosed == nil {
		onClosed = ConstantHandler{
			StatusCode: http.StatusServiceUnavailable,
		}
	}

	return &Gate{
		name:     name,
		onClosed: onClosed,
	}
}

// Name returns the name for this gate, which can be empty.  Typically,
// a name is useful when multiple gates are used within a single application.
func (g *Gate) Name() string {
	return g.name
}

// String returns a human-readable representation of this Gate.
func (g *Gate) String() string {
	stateText := gateClosedText
	if g.IsOpen() {
		stateText = gateOpenText
	}

	return fmt.Sprintf("gate[%s]:%s", g.name, stateText)
}

// IsOpen returns the current state of the gate: true for open and false for closed.
func (g *Gate) IsOpen() bool {
	return atomic.LoadUint32(&g.value) == gateOpen
}

// Open atomically opens this gate and invokes any registered OnOpen callbacks.  If the
// gate was already open, no callbacks are invoked since there was no state change.
func (g *Gate) Open() (opened bool) {
	// avoid the lock if no state change will occur
	if atomic.LoadUint32(&g.value) == gateOpen {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()
	opened = atomic.CompareAndSwapUint32(&g.value, gateClosed, gateOpen)
	if opened {
		for _, c := range g.onOpenCallbacks {
			c(g.name)
		}
	}

	return
}

// Close atomically closes this gate and invokes any registered OnClose callbacks.  If the
// gate was already closed, no callbacks are invoked since there was no state change.
func (g *Gate) Close() (closed bool) {
	// avoid the lock if no state change will occur
	if atomic.LoadUint32(&g.value) == gateClosed {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()
	closed = atomic.CompareAndSwapUint32(&g.value, gateOpen, gateClosed)
	if closed {
		for _, c := range g.onClosedCallbacks {
			c(g.name)
		}
	}

	return
}

// Append adds a hook to this gate's state.  If the given hook has no
// callbacks, this method does nothing.
//
// Immediately on calling the method, the appropriate callback (if set)
// is called to notify the infrastructure of the current state.  For example,
// if this method is called on an open gate and there is an OnOpen callback,
// then that OnOpen callback will be invoked.
func (g *Gate) Append(h GateHook) {
	if h.OnOpen == nil && h.OnClosed == nil {
		return // nop
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()

	open := g.IsOpen()
	if h.OnOpen != nil {
		g.onOpenCallbacks = append(g.onOpenCallbacks, h.OnOpen)
		if open {
			h.OnOpen(g.name)
		}
	}

	if h.OnClosed != nil {
		g.onClosedCallbacks = append(g.onClosedCallbacks, h.OnClosed)
		if !open {
			h.OnClosed(g.name)
		}
	}
}

// Then is a server middleware that guards its next http.Handler according
// to the gate's state.
func (g *Gate) Then(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if g.IsOpen() {
			next.ServeHTTP(response, request)
		}

		g.onClosed.ServeHTTP(response, request)
	})
}

// RoundTrip is a client middleware that guards its next http.RoundTripper
// according to the gate's state.  If next is nil, http.DefaultTransport is
// decorated instead.
//
// The onClosed http.Handler associated with this gate is not used for
// this middleware.  If the gate disallows the client request, a nil *http.Response
// together with a *GateClosedError are returned.
func (g *Gate) RoundTrip(next http.RoundTripper) http.RoundTripper {
	// keep a similar default behavior to other packages
	if next == nil {
		next = http.DefaultTransport
	}

	return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if g.IsOpen() {
			return next.RoundTrip(request)
		}

		return nil, &GateClosedError{Gate: g}
	})
}
