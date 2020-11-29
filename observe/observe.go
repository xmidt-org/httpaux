package observe

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// OnWriteHeader is a callback that is invoked once a handler either invokes WriteHeader
// or calls Write for the first time.  Note that if a handler never writes to the response
// body for any reason, including panicing, these callbacks will not be invoked.
//
// Use of this callback enables logging and metrics around things like how long a handler
// took to write headers.  Note that the status code passed to this callback is also
// available via the StatusCoder interface, so use of this callback is only in the niche
// cases where code needs to be aware of the point in time the header was written.
type OnWriteHeader func(int)

// StatusCoder is the interface implemented by an observable http.ResponseWriter.
type StatusCoder interface {
	// StatusCode returns the response code reported through WriteHeader.  Certain
	// methods cause WriteHeader to be called implicitly, with a status of http.StatusOK.
	//
	// If no status code has been written yet, this method returns zero (0).
	StatusCode() int
}

// ResponseBody is the interface implemented by an observable http.ResponseWriter
type ResponseBody interface {
	// ContentLength returns the count of bytes actually written with Write.  It does
	// not consult the Content-Length header.
	//
	// The count of bytes returned by this method is simply the current count of bytes
	// written so far.  If Write is called after this method, this method will return
	// a different value.
	ContentLength() int64
}

// Writer is the decorator interface for instrumented http.ResponseWriter instances.
// Instances of this interface are created with New to decorate an existing response writer.
type Writer interface {
	http.ResponseWriter
	StatusCoder
	ResponseBody

	// OnWriteHeader appends callbacks that are invoked when WriteHeader is called, whether
	// explicitly or implicitly due to calling methods like Flush.
	//
	// If the status code for the response has already been established, these callbacks
	// are invoked immediately.
	OnWriteHeader(...OnWriteHeader)
}

// responseWriterDecorator is the basic decorator for an http.ResponseWriter
type responseWriterDecorator struct {
	http.ResponseWriter
	headerWritten bool
	onWriteHeader []OnWriteHeader
	statusCode    int
	contentLength int64
}

func (rwd *responseWriterDecorator) StatusCode() int {
	return rwd.statusCode
}

func (rwd *responseWriterDecorator) ContentLength() int64 {
	return rwd.contentLength
}

func (rwd *responseWriterDecorator) OnWriteHeader(c ...OnWriteHeader) {
	if rwd.headerWritten {
		// no need to do any allocation
		for _, f := range c {
			f(rwd.statusCode)
		}
	} else {
		rwd.onWriteHeader = append(rwd.onWriteHeader, c...)
	}
}

func (rwd *responseWriterDecorator) Write(p []byte) (int, error) {
	if !rwd.headerWritten {
		// make sure any listener gets invoked properly
		rwd.WriteHeader(http.StatusOK)
	}

	c, err := rwd.ResponseWriter.Write(p)
	rwd.contentLength += int64(c)
	return c, err
}

func (rwd *responseWriterDecorator) WriteHeader(statusCode int) {
	if !rwd.headerWritten {
		rwd.statusCode = statusCode
		for _, f := range rwd.onWriteHeader {
			f(statusCode)
		}
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
	rfd.contentLength += c
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
var decorators = [16]func(*responseWriterDecorator) Writer{
	// 0000 - no decoration
	func(rwd *responseWriterDecorator) Writer {
		return rwd
	},
	// 0001 - http.Pusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			http.Pusher
		}{rwd, pusherDecorator{rwd}}
	},
	// 0010 - http.Flusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			http.Flusher
		}{rwd, flusherDecorator{rwd}}
	},
	// 0011 - http.Flusher, http.Pusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			http.Flusher
			http.Pusher
		}{rwd, flusherDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 0100 - http.Hijacker
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			http.Hijacker
		}{rwd, hijackerDecorator{rwd}}
	},
	// 0101 - http.Hijacker, http.Pusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			http.Hijacker
			http.Pusher
		}{rwd, hijackerDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 0110 - http.Hijacker, http.Flusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			http.Hijacker
			http.Flusher
		}{rwd, hijackerDecorator{rwd}, flusherDecorator{rwd}}
	},
	// 0111 - http.Hijacker, http.Flusher, http.Pusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			http.Hijacker
			http.Flusher
			http.Pusher
		}{rwd, hijackerDecorator{rwd}, flusherDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 1000 - io.ReaderFrom
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
		}{rwd, readerFromDecorator{rwd}}
	},
	// 1001 - io.ReaderFrom, http.Pusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Pusher
		}{rwd, readerFromDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 1010 - io.ReaderFrom, http.Flusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Flusher
		}{rwd, readerFromDecorator{rwd}, flusherDecorator{rwd}}
	},
	// 1011 - io.ReaderFrom, http.Flusher, http.Pusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Flusher
			http.Pusher
		}{rwd, readerFromDecorator{rwd}, flusherDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 1100 - io.ReaderFrom, http.Hijacker
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Hijacker
		}{rwd, readerFromDecorator{rwd}, hijackerDecorator{rwd}}
	},
	// 1101 - io.ReaderFrom, http.Hijacker, http.Pusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Hijacker
			http.Pusher
		}{rwd, readerFromDecorator{rwd}, hijackerDecorator{rwd}, pusherDecorator{rwd}}
	},
	// 1110 - io.ReaderFrom, http.Hijacker, http.Flusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Hijacker
			http.Flusher
		}{rwd, readerFromDecorator{rwd}, hijackerDecorator{rwd}, flusherDecorator{rwd}}
	},
	// 1111 - io.ReaderFrom, http.Hijacker, http.Flusher, http.Pusher
	func(rwd *responseWriterDecorator) Writer {
		return struct {
			*responseWriterDecorator
			io.ReaderFrom
			http.Hijacker
			http.Flusher
			http.Pusher
		}{rwd, readerFromDecorator{rwd}, hijackerDecorator{rwd}, flusherDecorator{rwd}, pusherDecorator{rwd}}
	},
}

// New decorates an http.ResponseWriter to produces a Writer
// If the delegate is already an Writer, it is returned as is.
//
// There are several interfaces in net/http that an http.ResponseWriter
// may optionally implement.  If the delegate implements any of those
// interfaces, the returned observable writer will as well.  The supported
// optional interfaces are:
//
//   - http.Pusher
//   - http.Flusher
//   - http.Hijacker
//   - io.ReaderFrom
func New(delegate http.ResponseWriter) Writer {
	if ow, ok := delegate.(Writer); ok {
		return ow
	}

	rwd := &responseWriterDecorator{
		ResponseWriter: delegate,
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
