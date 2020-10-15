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

func testObserveNoCallbacks(t *testing.T) {
	var (
		assert    = assert.New(t)
		handler   = ConstantHandler{StatusCode: 236}
		decorated = Observe().Then(handler)
	)

	assert.Equal(handler, decorated)
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

func testObserveStart(t *testing.T) {
	for _, testCase := range newDecorateCases() {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				request   = httptest.NewRequest("GET", "/", nil)
				decorated = Observe().Start(testCase.response, request)
			)

			require.NotNil(decorated)

			decorated.Header().Set("Test", "true")
			decorated.WriteHeader(298)
			decorated.Write([]byte("test"))

			assert.Equal(298, testCase.recorder.Code)
			assert.Equal(http.Header{"Test": {"true"}}, testCase.recorder.HeaderMap)
			assert.Equal("test", testCase.recorder.Body.String())

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

func testObserveCallbacks(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		beforeCalled bool
		before       = func(actual *http.Request) {
			beforeCalled = true
			assert.Equal(request, actual)
		}

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			assert.Equal(275, statusCode)
		}

		onWriteCalled bool
		onWrite       = func(c int64, err error) {
			onWriteCalled = true
			assert.Equal(int64(c), c)
			assert.NoError(err)
		}

		afterCalled bool
		after       = func(response http.ResponseWriter, actual *http.Request) {
			afterCalled = true
			assert.Equal(request, actual)
		}

		handler   = ConstantHandler{StatusCode: 275, Body: []byte("test")}
		decorated = Observe().
				Before(before).
				OnWriteHeader(onWriteHeader).
				OnWrite(onWrite).
				After(after).
				Then(handler)
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)
	assert.Equal(275, response.Code)
	assert.Equal("test", response.Body.String())

	assert.True(beforeCalled)
	assert.True(onWriteHeaderCalled)
	assert.True(onWriteCalled)
	assert.True(afterCalled)
}

func testObserveWriteWithoutWriteHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			assert.Equal(http.StatusOK, statusCode)
		}

		onWriteCalled bool
		onWrite       = func(c int64, err error) {
			onWriteCalled = true
			assert.Equal(int64(c), c)
			assert.NoError(err)
		}

		handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			// don't call WriteHeader.... the callback should still be called
			response.Write([]byte("test"))
		})

		decorated = Observe().
				OnWriteHeader(onWriteHeader).
				OnWrite(onWrite).
				Then(handler)
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal("test", response.Body.String())

	assert.True(onWriteHeaderCalled)
	assert.True(onWriteCalled)
}

func testObserveFlushWithoutWriteHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			assert.Equal(http.StatusOK, statusCode)
		}

		handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			// call Flush without calling WriteHeader
			response.(http.Flusher).Flush()
		})

		decorated = Observe().
				OnWriteHeader(onWriteHeader).
				Then(handler)
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.True(response.Flushed)

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

		request = httptest.NewRequest("GET", "/", nil)

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			assert.Equal(http.StatusOK, statusCode)
		}

		onWriteCalled bool
		onWrite       = func(c int64, err error) {
			onWriteCalled = true
			assert.Equal(int64(c), c)
			assert.NoError(err)
		}

		handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			// Use the io.ReaderFrom interface without calling WriteHeader
			// the callback should still get executed
			response.(io.ReaderFrom).ReadFrom(
				strings.NewReader("test"),
			)
		})

		decorated = Observe().
				OnWriteHeader(onWriteHeader).
				OnWrite(onWrite).
				Then(handler)
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal("test", verifier.readFrom.String())

	assert.True(onWriteHeaderCalled)
	assert.True(onWriteCalled)
}

func TestObserve(t *testing.T) {
	t.Run("Start", testObserveStart)
	t.Run("NoCallbacks", testObserveNoCallbacks)
	t.Run("Callbacks", testObserveCallbacks)
	t.Run("WriteWithoutWriteHeader", testObserveWriteWithoutWriteHeader)
	t.Run("FlushWithoutWriteHeader", testObserveFlushWithoutWriteHeader)
	t.Run("ReadFromWithoutWriteHeader", testObserveReadFromWithoutWriteHeader)
}
