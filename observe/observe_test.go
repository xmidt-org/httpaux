// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package observe

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
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

// newDecorateCases is factored out into its own function due to complexity.
// right now, only (1) test needs these cases
func newDecorateCases() (cases []decorateCase) {
	// no decoration
	{
		recorder := new(simpleRecorder)
		cases = append(cases, decorateCase{
			name:     "None",
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

type NewTestSuite struct {
	suite.Suite
}

func (suite *NewTestSuite) TestDecoration() {
	for _, testCase := range newDecorateCases() {
		suite.Run(testCase.name, func() {
			decorated := New(testCase.response)
			suite.Require().NotNil(decorated)

			decorated.Header().Set("Test", "true")
			decorated.WriteHeader(298)
			decorated.Write([]byte("test"))

			suite.Equal(298, testCase.recorder.Code)
			suite.Equal(http.Header{"Test": {"true"}}, testCase.recorder.HeaderMap)
			suite.Equal("test", testCase.recorder.Body.String())
			suite.Equal(298, decorated.StatusCode())
			suite.Equal(int64(4), decorated.ContentLength())

			if p, ok := decorated.(http.Pusher); ok {
				opts := &http.PushOptions{Method: "GET"}
				suite.NoError(p.Push("test", opts))
				suite.Equal("test", testCase.verifier.target)
				suite.Equal(opts, testCase.verifier.opts)
			}

			if f, ok := decorated.(http.Flusher); ok {
				f.Flush()
				suite.True(testCase.verifier.flushCalled)
			}

			if h, ok := decorated.(http.Hijacker); ok {
				h.Hijack()
				suite.True(testCase.verifier.hijackCalled)
			}

			if rf, ok := decorated.(io.ReaderFrom); ok {
				testCase.recorder.Body.Reset()
				c, err := rf.ReadFrom(strings.NewReader("read from"))
				suite.NoError(err)
				suite.Equal(len("read from"), int(c))

				suite.Equal("read from", testCase.verifier.readFrom.String())
			}
		})
	}
}

func (suite *NewTestSuite) TestNoDecoration() {
	var (
		response  = httptest.NewRecorder()
		decorated = New(response)
	)

	suite.Require().NotNil(decorated)

	again := New(decorated)
	suite.Require().NotNil(again)
	suite.True(decorated == again)
}

func (suite *NewTestSuite) TestWriteWithoutWriteHeader() {
	var (
		response = httptest.NewRecorder()

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			suite.Equal(http.StatusOK, statusCode)
		}

		decorated = New(response)
	)

	suite.Require().NotNil(decorated)
	decorated.OnWriteHeader(onWriteHeader)
	suite.False(onWriteHeaderCalled)

	decorated.Write([]byte("test"))
	suite.Equal(http.StatusOK, response.Code)
	suite.Equal("test", response.Body.String())
	suite.Equal(http.StatusOK, decorated.StatusCode())
	suite.True(onWriteHeaderCalled)

	// since WriteHeader was implicitly called, any new callbacks
	// should be invoked immediately
	onWriteHeaderCalled = false
	decorated.OnWriteHeader(onWriteHeader)
	suite.True(onWriteHeaderCalled)
}

func (suite *NewTestSuite) TestFlushWithoutWriteHeader() {
	var (
		response = httptest.NewRecorder()

		onWriteHeaderCalled bool
		onWriteHeader       = func(statusCode int) {
			onWriteHeaderCalled = true
			suite.Equal(http.StatusOK, statusCode)
		}

		decorated = New(response)
	)

	suite.Require().NotNil(decorated)
	decorated.OnWriteHeader(onWriteHeader)
	suite.False(onWriteHeaderCalled)

	decorated.(http.Flusher).Flush()
	suite.Equal(http.StatusOK, response.Code)
	suite.Equal(http.StatusOK, decorated.StatusCode())
	suite.True(response.Flushed)
	suite.True(onWriteHeaderCalled)

	// since WriteHeader was implicitly called, any new callbacks
	// should be invoked immediately
	onWriteHeaderCalled = false
	decorated.OnWriteHeader(onWriteHeader)
	suite.True(onWriteHeaderCalled)
}

func (suite *NewTestSuite) TestReadFromWithoutWriteHeader() {
	var (
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
			suite.Equal(http.StatusOK, statusCode)
		}

		decorated = New(response)
	)

	suite.Require().NotNil(decorated)
	decorated.OnWriteHeader(onWriteHeader)
	suite.False(onWriteHeaderCalled)

	decorated.(io.ReaderFrom).ReadFrom(strings.NewReader("test"))
	suite.Equal(http.StatusOK, response.Code)
	suite.Equal("test", verifier.readFrom.String())
	suite.Equal(http.StatusOK, decorated.StatusCode())
	suite.Equal(int64(4), decorated.ContentLength())
	suite.True(onWriteHeaderCalled)

	// since WriteHeader was implicitly called, any new callbacks
	// should be invoked immediately
	onWriteHeaderCalled = false
	decorated.OnWriteHeader(onWriteHeader)
	suite.True(onWriteHeaderCalled)
}

func TestNew(t *testing.T) {
	suite.Run(t, new(NewTestSuite))
}
