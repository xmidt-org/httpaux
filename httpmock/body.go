package httpmock

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrBodyClosed indicates that an attempt to read or close a Body, either request or response,
	// was made when the Body had already been closed.
	ErrBodyClosed = errors.New("The body has been closed")
)

// Closeable is implemented by anything that can report its closed state.
// *BodyReadCloser implements this interface.
type Closeable interface {
	// Closed reports whether Close() has been called on this instance.
	Closed() bool
}

// BodyReadCloser is a mock type that contains actual content but
// tracks whether the body has been closed.  NopCloser doesn't record
// whether Close was called, making it unusable for verifying proper
// http.Response handling.
type BodyReadCloser struct {
	buffer *bytes.Buffer
	closed bool
}

var _ io.Reader = (*BodyReadCloser)(nil)
var _ io.Closer = (*BodyReadCloser)(nil)
var _ Closeable = (*BodyReadCloser)(nil)

// Read delegates to the internal buffer to read whatever bytes it can.
// If this instance has had Close invoked on it, this method returns ErrBodyClosed.
func (brc *BodyReadCloser) Read(b []byte) (int, error) {
	if brc.closed {
		return 0, ErrBodyClosed
	}

	return brc.buffer.Read(b)
}

// Close marks this body as closed. Subsequent calls to this method
// will return ErrBodyClosed.
func (brc *BodyReadCloser) Close() error {
	if brc.closed {
		return ErrBodyClosed
	}

	brc.closed = true
	return nil
}

// Closed tests if this instance has had Close invoked on it.
func (brc *BodyReadCloser) Closed() bool {
	return brc.closed
}

// Closed returns true if the given body has been closed, false otherwise.
// This method panics if body does not implement Closed.
func Closed(body io.ReadCloser) bool {
	if c, ok := body.(Closeable); ok {
		return c.Closed()
	}

	panic("Body does not implemented Closeable")
}

// EmptyBody returns an empty body.
func EmptyBody() *BodyReadCloser {
	return &BodyReadCloser{
		buffer: new(bytes.Buffer),
	}
}

// BodyBytes returns a *BodyReadCloser backed by the given byte slice.
func BodyBytes(b []byte) *BodyReadCloser {
	return &BodyReadCloser{
		buffer: bytes.NewBuffer(b),
	}
}

// BodyString returns a *BodyReadCloser backed by the given string.
func BodyString(b string) *BodyReadCloser {
	return &BodyReadCloser{
		buffer: bytes.NewBufferString(b),
	}
}

// Bodyf produces a body with fmt.Sprintf.  This function is
// equivalent to:
//
//	BodyString(fmt.Sprintf(format, args...))
func Bodyf(format string, args ...interface{}) *BodyReadCloser {
	return BodyString(
		fmt.Sprintf(format, args...),
	)
}
