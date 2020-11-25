package httpaux

import (
	"fmt"
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

// Status is implemented by anything that can check an atomic boolean.
// All methods of this interface are safe for concurrent access.
type Status interface {
	// Name is an optional identifier for this atomic boolean.  No guarantees
	// as to uniqueness are made.  This value is completely up to client configuration.
	//
	// A non-empty name is often useful when an application uses multiple gates.  This
	// name can be used in log files, metrics, etc.
	Name() string

	// IsOpen checks if this instance is open and thus allowing traffic
	IsOpen() bool
}

// Control allows a gate to be open and closed atomically.  All methods of this interface
// are safe for concurrent access.
type Control interface {
	// Open raises this gate to allow traffic.  This method is atomic and idempotent.  It returns
	// true if there was a state change, false to indicate the gate was already open.
	Open() bool

	// Close lowers this gate to reject traffic.  This method is atomic and idempotent.  It returns
	// true if there was a state change, false to indicate the gate was already closed.
	Close() bool
}

// Interface represents a gate.  Instances are created via New.
type Interface interface {
	Status
	Control
}

// ClosedError is returned by any decorated infrastructure to indicate that the gate
// disallowed the client request.
type ClosedError struct {
	// Gate represents the gate instance that was closed at the time of the error.
	// Note that this gate may have been opened in the time that a caller waited on
	// the call to produce this error.
	Gate Status
}

// Error satisfies the error interface
func (ce *ClosedError) Error() string {
	return fmt.Sprintf("Gate [%s] closed", ce.Gate.Name())
}

// Callbacks is a convenient slice type for sequences of gate status callbacks
type Callbacks []func(Status)

// On invokes each callback with the given status
func (cb Callbacks) On(s Status) {
	for _, f := range cb {
		f(s)
	}
}

// Config describes all the various configurable settings for creating a Gate
type Config struct {
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
	//
	// No callback should ever panic, or later callbacks in this slice will be
	// short circuited.
	OnOpen Callbacks

	// OnClosed is the set of callbacks to invoke when a gate's state changes to closed.
	// These callbacks will also be invoked when a Gate is created if the Gate is
	// initially closed.
	//
	// No callback should ever panic, or later callbacks in this slice will be
	// short circuited.
	OnClosed Callbacks
}

// gate is the canonical implementation of both Status and Control
type gate struct {
	name string

	value     uint32
	stateLock sync.Mutex
	onOpen    Callbacks
	onClosed  Callbacks
}

// New produces a gate from a set of options.  The returned instance will be in
// the state indicated by Config.InitiallyClosed.
func New(c Config) Interface {
	g := &gate{
		name:     c.Name,
		onOpen:   append(Callbacks{}, c.OnOpen...),
		onClosed: append(Callbacks{}, c.OnClosed...),
	}

	if c.InitiallyClosed {
		g.value = gateClosed
	}

	// only invoke callbacks after everything is fully initialized
	if g.value == gateOpen {
		g.onOpen.On(g)
	} else {
		g.onClosed.On(g)
	}

	return g
}

func (g *gate) Name() string {
	return g.name
}

// String returns a human-readable representation of this Gate.
func (g *gate) String() string {
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

func (g *gate) IsOpen() bool {
	return atomic.LoadUint32(&g.value) == gateOpen
}

func (g *gate) Open() (opened bool) {
	if atomic.LoadUint32(&g.value) == gateOpen {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()
	opened = atomic.CompareAndSwapUint32(&g.value, gateClosed, gateOpen)
	if opened {
		g.onOpen.On(g)
	}

	return
}

func (g *gate) Close() (closed bool) {
	if atomic.LoadUint32(&g.value) == gateClosed {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()
	closed = atomic.CompareAndSwapUint32(&g.value, gateOpen, gateClosed)
	if closed {
		g.onClosed.On(g)
	}

	return
}
