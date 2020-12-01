package gate

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

// callbacks is a convenient slice type for sequences of gate status callbacks
type callbacks []func(Status)

func (cb callbacks) appendIfNotNil(f func(Status)) callbacks {
	if f != nil {
		return append(cb, f)
	}

	return cb
}

// on invokes each callback with the given status
func (cb callbacks) on(s Status) {
	for _, f := range cb {
		f(s)
	}
}

// Hook is a tuple of callbacks for gate state
type Hook struct {
	// OnOpen is invoked any time a gate is opened.  If a gate is open when this callback
	// is registered, it will be immediately invoked.
	//
	// Note: Callbacks should never modify the gate.  The Status instance passed to all callbacks
	// is not castable to Control.
	OnOpen func(Status)

	// OnClose is invoked any time a gate is closed.  If a gate is closed when this
	// callback is registered, it will be immediately invoked.
	//
	// Note: Callbacks should never modify the gate.  The Status instance passed to all callbacks
	// is not castable to Control.
	OnClosed func(Status)
}

// Hooks is a simple slice type for Hook instances
type Hooks []Hook

// Status is implemented by anything that can check an atomic boolean.
// All methods of this interface are safe for concurrent access.  None of
// the methods in this interface mutate the underlying gate.
//
// Note: although New returns a type that also implements this interface,
// Status instances passed to callbacks ARE NOT castable to Control.  This
// prevents modification of the gate by callbacks.
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
//
// All methods of this interface mutate the underlying gate and thus involve synchronization.
// In particular, Register is atomic with respect to Open/Close.
type Control interface {
	// Open raises this gate to allow traffic.  This method is atomic and idempotent.  It returns
	// true if there was a state change, false to indicate the gate was already open.
	Open() bool

	// Close lowers this gate to reject traffic.  This method is atomic and idempotent.  It returns
	// true if there was a state change, false to indicate the gate was already closed.
	Close() bool

	// Register adds a tuple of callbacks to this status instance.  If the given Hook
	// has no callbacks set, this method does nothing.
	//
	// Callbacks registered for the current state, e.g. OnOpen registered against an open gate,
	// will be immediately invoked prior to this method returning.
	Register(Hook)
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

	// Hooks is the optional set of preregistered callbacks when a gate is created with this
	// configuration.  Any empty hooks are silently ignored.
	//
	// Any callbacks that match the initial state of the gate, e.g. OnOpen when InitiallyClosed
	// is false, are immediately invoked before New returns.
	Hooks Hooks
}

// status is the internal Status implementation
type status struct {
	name  string
	value uint32
}

func (s *status) open() bool {
	return atomic.CompareAndSwapUint32(&s.value, gateClosed, gateOpen)
}

func (s *status) close() bool {
	return atomic.CompareAndSwapUint32(&s.value, gateOpen, gateClosed)
}

func (s *status) Name() string {
	return s.name
}

func (s *status) IsOpen() bool {
	return atomic.LoadUint32(&s.value) == gateOpen
}

// String returns a human-readable representation of this Gate.
func (s *status) String() string {
	stateText := gateClosedText
	if s.IsOpen() {
		stateText = gateOpenText
	}

	var b strings.Builder
	b.Grow(8 + len(stateText) + len(s.name))
	b.WriteString("gate[")
	b.WriteString(s.name)
	b.WriteString("]: ")
	b.WriteString(stateText)
	return b.String()
}

// gate is the canonical implementation of both Status and Control
type gate struct {
	// an embedded instance prevents callbacks from sidecasting to the Control interface
	*status
	stateLock sync.Mutex
	onOpen    callbacks
	onClosed  callbacks
}

// New produces a gate from a set of options.  The returned instance will be in
// the state indicated by Config.InitiallyClosed.
func New(c Config) Interface {
	g := &gate{
		status: &status{
			name: c.Name,
		},
	}

	for _, h := range c.Hooks {
		g.onOpen = g.onOpen.appendIfNotNil(h.OnOpen)
		g.onClosed = g.onClosed.appendIfNotNil(h.OnClosed)
	}

	// for consistency with Register, hold the lock while we invoke
	// any callbacks.  this will cause errant code to deadlock, as it should,
	// if Register is invoked on the same goroutine as a callback
	defer g.stateLock.Unlock()
	g.stateLock.Lock()

	if c.InitiallyClosed {
		// no need for atomic access here, as no other goroutines accessing
		// the gate under construction are possible at this point
		g.value = gateClosed
		g.onClosed.on(g.status)
	} else {
		g.onOpen.on(g.status)
	}

	return g
}

func (g *gate) Register(h Hook) {
	if h.OnOpen == nil && h.OnClosed == nil {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()

	isOpen := g.status.IsOpen()
	if h.OnOpen != nil {
		g.onOpen = append(g.onOpen, h.OnOpen)
		if isOpen {
			h.OnOpen(g.status)
		}
	}

	if h.OnClosed != nil {
		g.onClosed = append(g.onClosed, h.OnClosed)
		if isOpen {
			h.OnClosed(g.status)
		}
	}
}

func (g *gate) Open() (opened bool) {
	if g.status.IsOpen() {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()
	opened = g.status.open()
	if opened {
		g.onOpen.on(g.status)
	}

	return
}

func (g *gate) Close() (closed bool) {
	if !g.status.IsOpen() {
		return
	}

	defer g.stateLock.Unlock()
	g.stateLock.Lock()
	closed = g.status.close()
	if closed {
		g.onClosed.on(g.status)
	}

	return
}
