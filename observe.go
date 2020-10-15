package httpaux

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

// OB is a fluent builder for constructing an observable HTTP transaction.  This
// type should be constructed with Observe.
type OB struct {
	before        []BeforeHandle
	onWriteHeader []OnWriteHeader
	onWrite       []OnWrite
	after         []AfterHandle
}

func Observe() *OB {
	return new(OB)
}

func (ob *OB) Before(f ...BeforeHandle) *OB {
	ob.before = append(ob.before, f...)
	return ob
}

func (ob *OB) OnWriteHeader(f ...OnWriteHeader) *OB {
	ob.onWriteHeader = append(ob.onWriteHeader, f...)
	return ob
}

func (ob *OB) OnWrite(f ...OnWrite) *OB {
	ob.onWrite = append(ob.onWrite, f...)
	return ob
}

func (ob *OB) After(f ...AfterHandle) *OB {
	ob.after = append(ob.after, f...)
	return ob
}

func (ob *OB) invokeOnWriteHeader(statusCode int) {
	for _, f := range ob.onWriteHeader {
		f(statusCode)
	}
}

func (ob *OB) invokeOnWrite(written int64, err error) {
	for _, f := range ob.onWrite {
		f(written, err)
	}
}

func (ob *OB) decorate(delegate http.ResponseWriter) http.ResponseWriter {
	rwd := &responseWriterDecorator{
		ResponseWriter: delegate,
		onWriteHeader:  ob.invokeOnWriteHeader,
		onWrite:        ob.invokeOnWrite,
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

// Start begins an observable transaction.  The returned http.ResponseWriter should be
// passed to any handler code.
//
// Using Start (followed by End) directly is appropriate when the callbacks used change
// from request to request.
func (ob *OB) Start(response http.ResponseWriter, request *http.Request) http.ResponseWriter {
	for _, f := range ob.before {
		f(request)
	}

	return ob.decorate(response)
}

// End concludes an observable transaction.  The http.ResponseWriter passed to this method
// should be the same as that returned by Start.
//
// The typical use of this method is in a defer to ensure that callbacks get invoked
// after handler code has run.
func (ob *OB) End(response http.ResponseWriter, request *http.Request) {
	for _, f := range ob.after {
		f(response, request)
	}
}

// Then is a server middleware that decorates a given http.Handler with this observable's
// configuration.  This method is appropriate when the set of callbacks are the same
// for all requests.
func (ob *OB) Then(next http.Handler) http.Handler {
	// optimization: if there are no callbacks, don't decorate
	if len(ob.before) == 0 && len(ob.onWriteHeader) == 0 &&
		len(ob.onWrite) == 0 && len(ob.after) == 0 {
		return next
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		ow := ob.Start(response, request)
		defer ob.End(ow, request)
		next.ServeHTTP(ow, request)
	})
}
