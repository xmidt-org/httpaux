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

// GateStatus is implemented by anything that can check an atomic boolean.
// Gate implements this interface.
type GateStatus interface {
	// Name is an optional identifier for this atomic boolean.  No guarantees
	// as to uniqueness are made.  This value is completely up to client configuration.
	Name() string

	// IsOpen checks if this instance is open and thus allowing traffic
	IsOpen() bool
}

// GateClosedError is returned by any decorated infrastructure
// to indicate that the gate disallowed the client request.
type GateClosedError struct {
	// Gate is the gate instance that was closed at the time the attempt
	// to make a request was made.  Note that after this error returns, the
	// gate may be open.
	Gate GateStatus
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

	// InitiallyClosed indicates the state of a Gate when it is created.  The default
	// is to create a Gate that is open.  If this field is true, the Gate is created
	// in the closed state.
	InitiallyClosed bool

	// OnOpen is the set of callbacks to invoke when a gate's state changes to open.
	// These callbacks will also be invoked when a Gate is created if the Gate is
	// initially open.
	OnOpen []func(GateStatus)

	// OnClosed is the set of callbacks to invoke when a gate's state changes to closed.
	// These callbacks will also be invoked when a Gate is created if the Gate is
	// initially closed.
	OnClosed []func(GateStatus)
}

// Gate is an atomic boolean intended to control traffic in or out of a service.
// All methods of this type are safe for concurrent access.
type Gate struct {
	name string

	value     uint32
	stateLock sync.Mutex
	onOpen    []func(GateStatus)
	onClosed  []func(GateStatus)
}

// NewGate produces a Gate from a set of options.  The returned Gate will be in
// the state indicated by GateOptions.InitiallyClosed.
func NewGate(o GateOptions) *Gate {
	g := &Gate{
		name: o.Name,
	}

	if o.InitiallyClosed {
		g.value = gateClosed
	}

	// we want the Gate to be completely immutable in all ways except
	// for its atomic value.  so, make safe copies of callbacks.
	if len(o.OnOpen) > 0 {
		g.onOpen = append([]func(GateStatus){}, o.OnOpen...)
	}

	if len(o.OnClosed) > 0 {
		g.onClosed = append([]func(GateStatus){}, o.OnClosed...)
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

// GatedHandler is an http.Handler decorator that controls traffic to a given handler
// based upon the state of a gate.
type GatedHandler struct {
	// Next is the required http.Handler being gated.  This handler is only invoked
	// when the gate is open.
	Next http.Handler

	// Closed is the optional http.Handler to be invoked when the gate is closed.  This
	// handler may perform any custom logic desired.  If this field is unset, then
	// http.StatusServiceUnavailable is written to the response when the gate is closed.
	Closed http.Handler

	// GateStatus is the required controlling atomic boolean
	Gate GateStatus
}

// ServeHTTP invokes the Next handler if the gate status is open.  If the gate is closed and
// there is a Closed handler set, that handler is invoked.  Otherwise, http.StatusServiceUnavailable
// is written to the response.
func (gh GatedHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch {
	case gh.Gate.IsOpen():
		gh.Next.ServeHTTP(response, request)

	case gh.Closed != nil:
		gh.Closed.ServeHTTP(response, request)

	default:
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}

// GatedRoundTripper is an http.RoundTripper implementation that controls access to
// another round tripper based upon gate status.
type GatedRoundTripper struct {
	// Next is the required round tripper being controlled.  If this field is unset,
	Next http.RoundTripper

	// Closed is the optional round tripper that is invoked instead of Next when the
	// gate is closed.  If this method is not set, then a nil response and a GateClosedError
	// are returned by RoundTrip.
	Closed http.RoundTripper

	// Gate is the required atomic boolean which controls access to Next
	Gate GateStatus
}

// RoundTrip invokes the next round tripper if the gate status is open.  If the gate is
// closed and the Closed field is set, then the Closed round tripper is invoked.  Otherwise,
// a nil response and a GateClosedError are returned.
func (grt GatedRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	switch {
	case grt.Gate.IsOpen():
		return grt.Next.RoundTrip(request)

	case grt.Closed != nil:
		return grt.Closed.RoundTrip(request)

	default:
		return nil, &GateClosedError{Gate: grt.Gate}
	}
}
