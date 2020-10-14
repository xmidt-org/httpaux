package httpbuddy

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// BeforeHandle is a callback that is invoked prior to a handler being invoked.
// Implementations should not modify the request itself.
type BeforeHandle func(*http.Request)

// OnWriteHeader is a callback that is invoked once a handler either invokes WriteHeader
// or calls Write for the first time.  Note that if a handler never writes to the response
// body for any reason, including panicing, these callbacks will not be invoked.
type OnWriteHeader func(int)

// OnWrite is a callback that is invoked each time http.ResponseWriter.Write is called.
// This callback is passed the result of the Write method.
//
// Multiple calls to OnWrite should be expected.  If the total count of bytes written
// is desired, then implementations should keep a counter and update it with each callback.
type OnWrite func(int64, error)

// AfterHandle is a callback that is invoked once a handler has finished processing a request
type AfterHandle func(http.ResponseWriter, *http.Request)

// responseWriterDecorator is the basic decorator for an http.ResponseWriter
type responseWriterDecorator struct {
	http.ResponseWriter
	headerWritten bool
	onWriteHeader OnWriteHeader
	onWrite       OnWrite
}

func (rwd *responseWriterDecorator) Write(p []byte) (int, error) {
	if !rwd.headerWritten {
		// make sure any listener gets invoked properly
		rwd.WriteHeader(http.StatusOK)
	}

	c, err := rwd.ResponseWriter.Write(p)
	rwd.onWrite(int64(c), err)
	return c, err
}

func (rwd *responseWriterDecorator) WriteHeader(statusCode int) {
	if !rwd.headerWritten && rwd.onWriteHeader != nil {
		// only observe the first call
		rwd.onWriteHeader(statusCode)
	}

	// multiple WriteHeader calls is a bug, but we don't want
	// to hide that bug
	rwd.headerWritten = true
	rwd.ResponseWriter.WriteHeader(statusCode)
}

type flusherDecorator struct {
	*responseWriterDecorator
}

func (fd flusherDecorator) Flush() {
	if !fd.headerWritten {
		// make sure any listener gets invoked properly
		fd.WriteHeader(http.StatusOK)
	}

	fd.ResponseWriter.(http.Flusher).Flush()
}

type pusherDecorator struct {
	*responseWriterDecorator
}

func (pd pusherDecorator) Push(target string, opts *http.PushOptions) error {
	return pd.ResponseWriter.(http.Pusher).Push(target, opts)
}

type hijackerDecorator struct {
	*responseWriterDecorator
}

func (hd hijackerDecorator) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return hd.ResponseWriter.(http.Hijacker).Hijack()
}

type readerFromDecorator struct {
	*responseWriterDecorator
}

func (rfd readerFromDecorator) ReadFrom(r io.Reader) (int64, error) {
	if !rfd.headerWritten {
		// make sure any listener gets invoked properly
		rfd.WriteHeader(http.StatusOK)
	}

	c, err := rfd.ResponseWriter.(io.ReaderFrom).ReadFrom(r)
	rfd.onWrite(c, err)
	return c, err
}

const (
	// pusher is the bit mask for http.Pusher
	pusher = 1 << iota

	// flusher is the bit mask for http.Flusher
	flusher

	// hijacker is the bit mask for http.Hijacker
	hijacker

	// readerFrom is the bit mask for io.ReaderFrom
	readerFrom
)

// decorators is an array of factories that decorate the base responseWriterDecorator
// Each index in this array is a bit mask consisting of the types to decorate
var decorators = [16]func(*responseWriterDecorator) http.ResponseWriter{
	// 0000 - no decoration
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return rwd
	},
	// 0001 - http.Pusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			http.Pusher
		}{rwd, pusherDecorator{rwd}}
	},
	// 0010 - http.Flusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			http.Flusher
		}{rwd, flusherDecorator{rwd}}
	},
	// 0011 - http.Flusher, http.Pusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			http.Flusher
			http.Pusher
		}{rwd, flusherDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 0100 - http.Hijacker
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			http.Hijacker
		}{rwd, hijackerDecorator{rwd}}
	},
	// 0101 - http.Hijacker, http.Pusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			http.Hijacker
			http.Pusher
		}{rwd, hijackerDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 0110 - http.Hijacker, http.Flusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			http.Hijacker
			http.Flusher
		}{rwd, hijackerDecorator{rwd}, flusherDecorator{rwd}}
	},
	// 0111 - http.Hijacker, http.Flusher, http.Pusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			http.Hijacker
			http.Flusher
			http.Pusher
		}{rwd, hijackerDecorator{rwd}, flusherDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 1000 - io.ReaderFrom
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
		}{rwd, readerFromDecorator{rwd}}
	},
	// 1001 - io.ReaderFrom, http.Pusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Pusher
		}{rwd, readerFromDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 1010 - io.ReaderFrom, http.Flusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Flusher
		}{rwd, readerFromDecorator{rwd}, flusherDecorator{rwd}}
	},
	// 1011 - io.ReaderFrom, http.Flusher, http.Pusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Flusher
			http.Pusher
		}{rwd, readerFromDecorator{rwd}, flusherDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 1100 - io.ReaderFrom, http.Hijacker
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Hijacker
		}{rwd, readerFromDecorator{rwd}, hijackerDecorator{rwd}}
	},
	// 1101 - io.ReaderFrom, http.Hijacker, http.Pusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Hijacker
			http.Pusher
		}{rwd, readerFromDecorator{rwd}, hijackerDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 1110 - io.ReaderFrom, http.Hijacker, http.Flusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Hijacker
			http.Flusher
		}{rwd, readerFromDecorator{rwd}, hijackerDecorator{rwd}, flusherDecorator{rwd}}
	},
	// 1111 - io.ReaderFrom, http.Hijacker, http.Flusher, http.Pusher
	func(rwd *responseWriterDecorator) http.ResponseWriter {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Hijacker
			http.Flusher
			http.Pusher
		}{rwd, readerFromDecorator{rwd}, hijackerDecorator{rwd}, flusherDecorator{rwd}, pusherDecorator{rwd}}
	},
}

// TB is a fluent builder for creating a server middleware that creates TrackingWriter
// objects for downstream handlers.  This type should be instantiated with Track.
type TB struct {
	before        []BeforeHandle
	onWriteHeader []OnWriteHeader
	onWrite       []OnWrite
	after         []AfterHandle
}

// Track starts a fluent builder chain for building a server middleware for tracking
// serverside request processing.
func Track() *TB {
	return new(TB)
}

// Before appends callbacks that are invoked before the request is actually handled
func (tb *TB) Before(f ...BeforeHandle) *TB {
	tb.before = append(tb.before, f...)
	return tb
}

// OnWriteHeader appends callbacks that are invoked whenever http.ResponseWriter.WriterHeader
// is invoked.  Note that if a decorated never calls WriteHeader or writes to the request body,
// these callbacks will not be called.  This includes panicing before WriteHeader is called.
func (tb *TB) OnWriteHeader(f ...OnWriteHeader) *TB {
	tb.onWriteHeader = append(tb.onWriteHeader, f...)
	return tb
}

// OnWrite appends callbacks that are invoked each time http.ResponseWriter.Write is invoked.
// If http.Handler code never writes to a response, these callbacks are never invoked.
func (tb *TB) OnWrite(f ...OnWrite) *TB {
	tb.onWrite = append(tb.onWrite, f...)
	return tb
}

// After appends callbacks that are invoked after the request has been handled
func (tb *TB) After(f ...AfterHandle) *TB {
	tb.after = append(tb.after, f...)
	return tb
}

// Then returns a server middleware that decorates handlers using this builder's
// configuration.  Future changes to this builder will not be reflected in the
// returned http.Handler.
func (tb *TB) Then(next http.Handler) http.Handler {
	return &trackingHandler{
		next:          next,
		before:        append([]BeforeHandle{}, tb.before...),
		onWriteHeader: append([]OnWriteHeader{}, tb.onWriteHeader...),
		onWrite:       append([]OnWrite{}, tb.onWrite...),
		after:         append([]AfterHandle{}, tb.after...),
	}
}

// trackingHandler is a handler decorator that handles the lifecycle of
// a TrackingWriter together with the various callbacks
type trackingHandler struct {
	next          http.Handler
	before        []BeforeHandle
	onWriteHeader []OnWriteHeader
	onWrite       []OnWrite
	after         []AfterHandle
}

func (th *trackingHandler) invokeBefore(r *http.Request) {
	for _, f := range th.before {
		f(r)
	}
}

func (th *trackingHandler) invokeOnWriteHeader(statusCode int) {
	for _, f := range th.onWriteHeader {
		f(statusCode)
	}
}

func (th *trackingHandler) invokeOnWrite(written int64, err error) {
	for _, f := range th.onWrite {
		f(written, err)
	}
}

func (th *trackingHandler) invokeAfter(rw http.ResponseWriter, r *http.Request) {
	for _, f := range th.after {
		f(rw, r)
	}
}

func (th *trackingHandler) newTrackingWriter(delegate http.ResponseWriter) http.ResponseWriter {
	rwd := &responseWriterDecorator{
		ResponseWriter: delegate,
		onWriteHeader:  th.invokeOnWriteHeader,
		onWrite:        th.invokeOnWrite,
	}

	mask := 0
	if _, ok := delegate.(http.Pusher); ok {
		mask += pusher
	}

	if _, ok := delegate.(http.Flusher); ok {
		mask += flusher
	}

	if _, ok := delegate.(http.Hijacker); ok {
		mask += hijacker
	}

	if _, ok := delegate.(io.ReaderFrom); ok {
		mask += readerFrom
	}

	return decorators[mask](rwd)
}

func (th *trackingHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	th.invokeBefore(request)
	tw := th.newTrackingWriter(response)

	// don't pass the tracking write to after, as any misuse of the API
	// would result in callbacks
	defer th.invokeAfter(response, request)

	th.next.ServeHTTP(tw, request)
}
