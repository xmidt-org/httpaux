package httpaux

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simpleRecorder is an http.ResponseWriter that implements none of the
// other optional interfaces
type simpleRecorder struct {
	Code      int
	HeaderMap http.Header
	Body      bytes.Buffer
}

func (sr *simpleRecorder) Header() http.Header {
	if sr.HeaderMap == nil {
		sr.HeaderMap = http.Header{}
	}

	return sr.HeaderMap
}

func (sr *simpleRecorder) WriteHeader(c int) {
	sr.Code = c
}

func (sr *simpleRecorder) Write(p []byte) (int, error) {
	return sr.Body.Write(p)
}

// decorateVerifier is used to check that the decoration logic
// properly invokes the delegate
type decorateVerifier struct {
	flushCalled bool

	target string
	opts   *http.PushOptions

	hijackCalled bool

	readFrom bytes.Buffer
}

type testFlusher struct {
	*decorateVerifier
}

func (tf *testFlusher) Flush() {
	tf.flushCalled = true
}

type testPusher struct {
	*decorateVerifier
}

func (tp *testPusher) Push(target string, opts *http.PushOptions) error {
	tp.target = target
	tp.opts = opts
	return nil
}

type testHijacker struct {
	*decorateVerifier
}

func (th *testHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	th.hijackCalled = true
	return nil, nil, nil
}

type testReaderFrom struct {
	*decorateVerifier
}

func (trf *testReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(&trf.readFrom, r)
}

type decorateCase struct {
	name     string
	verifier *decorateVerifier
	recorder *simpleRecorder
	response http.ResponseWriter
}

// newDecorateCases builds are test data records, since we can't statically define them.
// these have to be built programmatically
func newDecorateCases() (cases []decorateCase) {
	// no decoration
	{
		recorder := new(simpleRecorder)
		cases = append(cases, decorateCase{
			name:     "NoDecoration",
			verifier: new(decorateVerifier),
			recorder: recorder,
			response: recorder,
		})
	}

	// pusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "Pusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				http.Pusher
			}{recorder, &testPusher{verifier}},
		})
	}

	// flusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "Flusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				http.Flusher
			}{recorder, &testFlusher{verifier}},
		})
	}

	// flusher + pusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "Flusher,Pusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				http.Flusher
				http.Pusher
			}{recorder, &testFlusher{verifier}, &testPusher{verifier}},
		})
	}

	// hijacker
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "Hijacker",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				http.Hijacker
			}{recorder, &testHijacker{verifier}},
		})
	}

	// hijacker + pusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "Hijacker,Pusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				http.Hijacker
				http.Pusher
			}{recorder, &testHijacker{verifier}, &testPusher{verifier}},
		})
	}

	// hijacker + flusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "Hijacker,Flusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				http.Hijacker
				http.Flusher
			}{recorder, &testHijacker{verifier}, &testFlusher{verifier}},
		})
	}

	// hijacker + flusher + pusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "Hijacker,Flusher,Pusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				http.Hijacker
				http.Flusher
				http.Pusher
			}{recorder, &testHijacker{verifier}, &testFlusher{verifier}, &testPusher{verifier}},
		})
	}

	// readerFrom
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "ReaderFrom",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				io.ReaderFrom
			}{recorder, &testReaderFrom{verifier}},
		})
	}

	// readerFrom + pusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "ReaderFrom,Pusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				io.ReaderFrom
				http.Pusher
			}{recorder, &testReaderFrom{verifier}, &testPusher{verifier}},
		})
	}

	// readerFrom + flusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "ReaderFrom,Flusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				io.ReaderFrom
				http.Flusher
			}{recorder, &testReaderFrom{verifier}, &testFlusher{verifier}},
		})
	}

	// readerFrom + flusher + pusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "ReaderFrom,Flusher,Pusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				io.ReaderFrom
				http.Flusher
				http.Pusher
			}{recorder, &testReaderFrom{verifier}, &testFlusher{verifier}, &testPusher{verifier}},
		})
	}

	// readerFrom + hijacker
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "ReaderFrom,Hijacker",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				io.ReaderFrom
				http.Hijacker
			}{recorder, &testReaderFrom{verifier}, &testHijacker{verifier}},
		})
	}

	// readerFrom + hijacker + pusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "ReaderFrom,Hijacker,Pusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				io.ReaderFrom
				http.Hijacker
				http.Pusher
			}{recorder, &testReaderFrom{verifier}, &testHijacker{verifier}, &testPusher{verifier}},
		})
	}

	// readerFrom + hijacker + flusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "ReaderFrom,Hijacker,Flusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				io.ReaderFrom
				http.Hijacker
				http.Flusher
			}{recorder, &testReaderFrom{verifier}, &testHijacker{verifier}, &testFlusher{verifier}},
		})
	}

	// readerFrom + hijacker + flusher + pusher
	{
		recorder := new(simpleRecorder)
		verifier := new(decorateVerifier)
		cases = append(cases, decorateCase{
			name:     "ReaderFrom,Hijacker,Flusher,Pusher",
			verifier: verifier,
			recorder: recorder,
			response: struct {
				http.ResponseWriter
				io.ReaderFrom
				http.Hijacker
				http.Flusher
				http.Pusher
			}{recorder, &testReaderFrom{verifier}, &testHijacker{verifier}, &testFlusher{verifier}, &testPusher{verifier}},
		})
	}

	return
}

func testObserveDecoration(t *testing.T) {
	for _, testCase := range newDecorateCases() {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				decorated = Observe(testCase.response)
			)

			require.NotNil(decorated)

			decorated.Header().Set("Test", "true")
			decorated.WriteHeader(298)
			decorated.Write([]byte("test"))

			assert.Equal(298, testCase.recorder.Code)
			assert.Equal(http.Header{"Test": {"true"}}, testCase.recorder.HeaderMap)
			assert.Equal("test", testCase.recorder.Body.String())
			assert.Equal(298, decorated.StatusCode())
			assert.Equal(int64(4), decorated.ContentLength())

			if p, ok := decorated.(http.Pusher); ok {
				opts := &http.PushOptions{Method: "GET"}
				assert.NoError(p.Push("test", opts))
				assert.Equal("test", testCase.verifier.target)
				assert.Equal(opts, testCase.verifier.opts)
			}

			if f, ok := decorated.(http.Flusher); ok {
				f.Flush()
				assert.True(testCase.verifier.flushCalled)
			}

			if h, ok := decorated.(http.Hijacker); ok {
				h.Hijack()
				assert.True(testCase.verifier.hijackCalled)
			}

			if rf, ok := decorated.(io.ReaderFrom); ok {
				testCase.recorder.Body.Reset()
				c, err := rf.ReadFrom(strings.NewReader("read from"))
				assert.NoError(err)
				assert.Equal(len("read from"), int(c))

				assert.Equal("read from", testCase.verifier.readFrom.String())
			}
		})
	}
}

func testObserveNoDecoration(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		response  = httptest.NewRecorder()
		decorated = Observe(response)
	)

	require.NotNil(decorated)

	again := Observe(decorated)
	require.NotNil(again)
	assert.True(decorated == again)
}

func testObserveWriteWithoutWriteHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		response = httptest.NewRecorder()

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			assert.Equal(http.StatusOK, statusCode)
		}

		decorated = Observe(response)
	)

	require.NotNil(decorated)
	decorated.OnWriteHeader(onWriteHeader)
	assert.False(onWriteHeaderCalled)

	decorated.Write([]byte("test"))
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal("test", response.Body.String())
	assert.Equal(http.StatusOK, decorated.StatusCode())
	assert.True(onWriteHeaderCalled)

	// since WriteHeader was implicitly called, any new callbacks
	// should be invoked immediately
	onWriteHeaderCalled = false
	decorated.OnWriteHeader(onWriteHeader)
	assert.True(onWriteHeaderCalled)
}

func testObserveFlushWithoutWriteHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		response = httptest.NewRecorder()

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			assert.Equal(http.StatusOK, statusCode)
		}

		decorated = Observe(response)
	)

	require.NotNil(decorated)
	decorated.OnWriteHeader(onWriteHeader)
	assert.False(onWriteHeaderCalled)

	decorated.(http.Flusher).Flush()
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal(http.StatusOK, decorated.StatusCode())
	assert.True(response.Flushed)
	assert.True(onWriteHeaderCalled)

	// since WriteHeader was implicitly called, any new callbacks
	// should be invoked immediately
	onWriteHeaderCalled = false
	decorated.OnWriteHeader(onWriteHeader)
	assert.True(onWriteHeaderCalled)
}

func testObserveReadFromWithoutWriteHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		verifier = new(decorateVerifier)
		response = struct {
			*simpleRecorder
			*testReaderFrom
		}{
			new(simpleRecorder),
			&testReaderFrom{verifier},
		}

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			assert.Equal(http.StatusOK, statusCode)
		}

		decorated = Observe(response)
	)

	require.NotNil(decorated)
	decorated.OnWriteHeader(onWriteHeader)
	assert.False(onWriteHeaderCalled)

	decorated.(io.ReaderFrom).ReadFrom(strings.NewReader("test"))
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal("test", verifier.readFrom.String())
	assert.Equal(http.StatusOK, decorated.StatusCode())
	assert.Equal(int64(4), decorated.ContentLength())
	assert.True(onWriteHeaderCalled)

	// since WriteHeader was implicitly called, any new callbacks
	// should be invoked immediately
	onWriteHeaderCalled = false
	decorated.OnWriteHeader(onWriteHeader)
	assert.True(onWriteHeaderCalled)
}

func TestObserve(t *testing.T) {
	t.Run("Decoration", testObserveDecoration)
	t.Run("NoDecoration", testObserveNoDecoration)
	t.Run("WriteWithoutWriteHeader", testObserveWriteWithoutWriteHeader)
	t.Run("FlushWithoutWriteHeader", testObserveFlushWithoutWriteHeader)
	t.Run("ReadFromWithoutWriteHeader", testObserveReadFromWithoutWriteHeader)
}
