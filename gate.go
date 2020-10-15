package httpaux

import (
	"fmt"
	"net/http"
	"strings"
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

// GateOptions describes all the various configurable settings for creating a Gate
type GateOptions struct {
	// Name is an optional identifier for this gate.  The Gate itself does not make
	// use of this value.  It's purely for distinguishing gates when an application
	// uses more than one (1) gate.
	Name string

	// ClosedHandler is the optional http.Handler that handles requests when
	// the gate is closed.  If this field is unset, then the only response information
	// written is a status code of http.StatusServiceUnavailable.
	ClosedHandler http.Handler

	// InitiallyClosed indicates the state of a Gate when it is created.  The default
	// is to create a Gate that is open.  If this field is true, the Gate is created
	// in the closed state.
	InitiallyClosed bool

	// OnOpen is the set of callbacks to invoke when a gate's state changes to open.
	// These callbacks will also be invoked when a Gate is created if the Gate is
	// initially open.
	OnOpen []func(*Gate)

	// OnClosed is the set of callbacks to invoke when a gate's state changes to closed.
	// These callbacks will also be invoked when a Gate is created if the Gate is
	// initially closed.
	OnClosed []func(*Gate)
}

// Gate is an atomic boolean that controls access to http.Handlers and http.RoundTrippers.
// All methods of this type are safe for concurrent access.
//
// A Gate can be observed via the Append method and supplying callbacks for state.  These
// callbacks are useful for integrating logging, metrics, health checks, etc.
type Gate struct {
	name          string
	closedHandler http.Handler

	value     uint32
	stateLock sync.Mutex
	onOpen    []func(*Gate)
	onClosed  []func(*Gate)
}

// NewGate produces a Gate from a set of options.  The returned Gate will be in
// the state indicated by GateOptions.InitiallyClosed.
func NewGate(o GateOptions) *Gate {
	g := &Gate{
		name: o.Name,
	}

	if o.ClosedHandler != nil {
		g.closedHandler = o.ClosedHandler
	} else {
		g.closedHandler = ConstantHandler{
			StatusCode: http.StatusServiceUnavailable,
		}
	}

	if o.InitiallyClosed {
		g.value = gateClosed
	}

	// we want the Gate to be completely immutable in all ways except
	// for its atomic value.  so, make safe copies of callbacks.
	if len(o.OnOpen) > 0 {
		g.onOpen = append([]func(*Gate){}, o.OnOpen...)
	}

	if len(o.OnClosed) > 0 {
		g.onClosed = append([]func(*Gate){}, o.OnClosed...)
	}

	// only invoke callbacks after everything is fully initialized
	if g.value == gateOpen {
		for _, f := range g.onOpen {
			f(g)
		}
	} else {
		for _, f := range g.onClosed {
			f(g)
		}
	}

	return g
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

	var b strings.Builder
	b.Grow(8 + len(stateText) + len(g.name))
	b.WriteString("gate[")
	b.WriteString(g.name)
	b.WriteString("]: ")
	b.WriteString(stateText)
	return b.String()
}

// IsOpen returns the current state of the gate: true for open and false for closed.
func (g *Gate) IsOpen() bool {
	return atomic.LoadUint32(&g.value) == gateOpen
}

// Open atomically opens this gate and invokes any registered OnOpen callbacks.  If the
// gate was already open, no callbacks are invoked since there was no state change.
func (g *Gate) Open() (opened bool) {
	if atomic.LoadUint32(&g.value) == gateOpen {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()
	opened = atomic.CompareAndSwapUint32(&g.value, gateClosed, gateOpen)
	if opened {
		for _, f := range g.onOpen {
			f(g)
		}
	}

	return
}

// Close atomically closes this gate and invokes any registered OnClose callbacks.  If the
// gate was already closed, no callbacks are invoked since there was no state change.
func (g *Gate) Close() (closed bool) {
	if atomic.LoadUint32(&g.value) == gateClosed {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()
	closed = atomic.CompareAndSwapUint32(&g.value, gateOpen, gateClosed)
	if closed {
		for _, f := range g.onClosed {
			f(g)
		}
	}

	return
}

// Then is a server middleware that guards its next http.Handler according
// to the gate's state.
func (g *Gate) Then(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if g.IsOpen() {
			next.ServeHTTP(response, request)
		}

		g.closedHandler.ServeHTTP(response, request)
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
