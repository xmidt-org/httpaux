package httpmock

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// RequestCheckerTestSuite is the sole suite for all the strategies
// in this package, even the ones that only matchers or asserters.
type RequestCheckerTestSuite struct {
	suite.Suite
}

func (suite *RequestCheckerTestSuite) TestRequestMatcherFunc() {
	expected := new(http.Request)
	var called bool

	rmf := RequestMatcherFunc(func(actual *http.Request) bool {
		suite.True(expected == actual)
		called = true
		return true
	})

	suite.True(rmf.Match(expected))
	suite.True(called)
}

func (suite *RequestCheckerTestSuite) TestRequestAsserterFunc() {
	expectedAssert := new(assert.Assertions)
	expectedRequest := new(http.Request)
	var called bool

	raf := RequestAsserterFunc(func(actualAssert *assert.Assertions, actualRequest *http.Request) {
		suite.True(expectedAssert == actualAssert)
		suite.True(expectedRequest == actualRequest)
		called = true
	})

	raf.Assert(expectedAssert, expectedRequest)
	suite.True(called)
}

func (suite *RequestCheckerTestSuite) TestNopRequestChecker() {
	suite.Run("Match", func() {
		suite.True(
			NopRequestChecker{}.Match(nil),
		)
	})

	suite.Run("Assert", func() {
		t := wrapTestingT(suite.T())
		NopRequestChecker{}.Assert(assert.New(t), nil)
		suite.Zero(t.Logs)
		suite.Zero(t.Errors)
		suite.Zero(t.Failures)
	})
}

func (suite *RequestCheckerTestSuite) TestMethods() {
	suite.Run("Pass", func() {
		testData := []struct {
			checker RequestChecker
			request *http.Request
		}{
			{
				checker: Methods("GET"),
				request: &http.Request{Method: "GET"},
			},
			{
				checker: Methods("POST", "PUT"),
				request: &http.Request{Method: "PUT"},
			},
			{
				checker: Methods("POST", "PUT", "GET"),
				request: &http.Request{Method: "POST"},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				suite.True(
					record.checker.Match(record.request),
				)

				t := wrapTestingT(suite.T())
				record.checker.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Zero(t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})

	suite.Run("Fail", func() {
		testData := []struct {
			checker RequestChecker
			request *http.Request
		}{
			{
				checker: Methods(),
				request: &http.Request{Method: "doesn't matter"},
			},
			{
				checker: Methods("GET"),
				request: &http.Request{Method: "POST"},
			},
			{
				checker: Methods("POST", "PUT"),
				request: &http.Request{Method: "PATCH"},
			},
			{
				checker: Methods("POST", "PUT", "GET"),
				request: &http.Request{Method: "DELETE"},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				suite.False(
					record.checker.Match(record.request),
				)

				t := wrapTestingT(suite.T())
				record.checker.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Equal(1, t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})
}

func (suite *RequestCheckerTestSuite) TestPath() {
	suite.Run("Pass", func() {
		p := Path("/test")
		suite.True(
			p.Match(&http.Request{URL: &url.URL{Path: "/test"}}),
		)

		t := wrapTestingT(suite.T())
		p.Assert(assert.New(t), &http.Request{URL: &url.URL{Path: "/test"}})
		suite.Zero(t.Logs)
		suite.Zero(t.Errors)
		suite.Zero(t.Failures)
	})

	suite.Run("Fail", func() {
		testData := []struct {
			checker RequestChecker
			request *http.Request
		}{
			{
				checker: Path("/test"),
				request: new(http.Request), // no URL
			},
			{
				checker: Path("/test"),
				request: &http.Request{URL: &url.URL{Path: "/doesnotmatch"}},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				suite.False(
					record.checker.Match(record.request),
				)

				t := wrapTestingT(suite.T())
				record.checker.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Equal(1, t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})
}

func (suite *RequestCheckerTestSuite) TestHeader() {
	suite.Run("Pass", func() {
		testData := []struct {
			checker RequestChecker
			request *http.Request
		}{
			{
				checker: Header("Test"),
				request: &http.Request{
					Header: http.Header{},
				},
			},
			{
				checker: Header("Test", "value1"),
				request: &http.Request{
					Header: http.Header{
						"Test": {"value1"},
					},
				},
			},
			{
				checker: Header("Test", "value1", "value2"),
				request: &http.Request{
					Header: http.Header{
						"Test": {"value2", "value1"},
					},
				},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				suite.True(
					record.checker.Match(record.request),
				)

				t := wrapTestingT(suite.T())
				record.checker.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Zero(t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})

	suite.Run("Fail", func() {
		testData := []struct {
			checker RequestChecker
			request *http.Request
		}{
			{
				checker: Header("Test"),
				request: &http.Request{
					Header: http.Header{
						"Test": {"something"},
					},
				},
			},
			{
				checker: Header("Test", "value1"),
				request: &http.Request{
					Header: http.Header{
						"Test": {"doesnotmatch"},
					},
				},
			},
			{
				checker: Header("Test", "value1", "value2"),
				request: &http.Request{
					Header: http.Header{
						"Test": {"value1"},
					},
				},
			},
			{
				checker: Header("Test", "value1", "value2"),
				request: &http.Request{
					Header: http.Header{
						"Test": {"value1", "value3"},
					},
				},
			},
			{
				checker: Header("Test", "value1", "value2"),
				request: &http.Request{
					Header: http.Header{
						"Test": {"value1", "value3", "value0"},
					},
				},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				suite.False(
					record.checker.Match(record.request),
				)

				t := wrapTestingT(suite.T())
				record.checker.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Equal(1, t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})
}

func (suite *RequestCheckerTestSuite) TestBody() {
	suite.Run("Pass", func() {
		testData := []struct {
			asserter RequestAsserter
			request  *http.Request
		}{
			{
				asserter: Body(""),
				request:  new(http.Request), // nil body matches empty expected
			},
			{
				asserter: Body("test text"),
				request: &http.Request{
					Body: BodyString("test text"),
				},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				t := wrapTestingT(suite.T())
				record.asserter.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Zero(t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})

	suite.Run("Fail", func() {
		testData := []struct {
			asserter RequestAsserter
			request  *http.Request
		}{
			{
				asserter: Body("cannot be empty"),
				request:  new(http.Request), // nil Body
			},
			{
				asserter: Body("cannot be empty"),
				request: &http.Request{
					Body: BodyString(""),
				},
			},
			{
				asserter: Body(""),
				request: &http.Request{
					Body: BodyString("is not empty"),
				},
			},
			{
				asserter: Body("expected"),
				request: &http.Request{
					Body: BodyString("... and this doesn't match"),
				},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				t := wrapTestingT(suite.T())
				record.asserter.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Equal(1, t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})

	suite.Run("ReadError", func() {
		reader, writer := io.Pipe()
		defer reader.Close()

		go func() {
			writer.Write([]byte("test"))
			writer.CloseWithError(errors.New("expected"))
		}()

		request := &http.Request{
			Body: io.NopCloser(reader),
		}

		t := wrapTestingT(suite.T())
		b := Body("test")
		b.Assert(assert.New(t), request)

		suite.Zero(t.Logs)
		suite.Equal(1, t.Errors)
		suite.Zero(t.Failures)
	})
}

func (suite *RequestCheckerTestSuite) TestBodyJSON() {
	suite.Run("Pass", func() {
		testData := []struct {
			asserter RequestAsserter
			request  *http.Request
		}{
			{
				asserter: BodyJSON(""),
				request:  new(http.Request), // nil body matches empty expected
			},
			{
				asserter: BodyJSON(`{"value1": 1, "value3": {"a": "b"}, "value2": "hello"}`),
				request: &http.Request{
					Body: BodyString(`{"value2": "hello", "value1": 1, "value3": {"a": "b"}}`),
				},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				t := wrapTestingT(suite.T())
				record.asserter.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Zero(t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})

	suite.Run("Fail", func() {
		testData := []struct {
			asserter RequestAsserter
			request  *http.Request
		}{
			{
				asserter: BodyJSON(`{"value1": "won't match nil"}`),
				request:  new(http.Request), // nil Body
			},
			{
				asserter: BodyJSON(`{"value1": "cannot be empty"}`),
				request: &http.Request{
					Body: BodyString(""),
				},
			},
			{
				asserter: BodyJSON(""),
				request: &http.Request{
					Body: BodyString(`{"value1": "isn't empty"}`),
				},
			},
			{
				asserter: BodyJSON(`{"name": "Simon", "age": 75}`),
				request: &http.Request{
					Body: BodyString(`{"this": "does not match"}`),
				},
			},
		}

		for i, record := range testData {
			suite.Run(strconv.Itoa(i), func() {
				t := wrapTestingT(suite.T())
				record.asserter.Assert(assert.New(t), record.request)
				suite.Zero(t.Logs)
				suite.Equal(1, t.Errors)
				suite.Zero(t.Failures)
			})
		}
	})

	suite.Run("ReadError", func() {
		reader, writer := io.Pipe()
		defer reader.Close()

		go func() {
			writer.Write([]byte("test"))
			writer.CloseWithError(errors.New("expected"))
		}()

		request := &http.Request{
			Body: io.NopCloser(reader),
		}

		t := wrapTestingT(suite.T())
		b := BodyJSON(`{"this": "won't matter"}`)
		b.Assert(assert.New(t), request)

		suite.Zero(t.Logs)
		suite.Equal(1, t.Errors)
		suite.Zero(t.Failures)
	})
}

func TestRequestChecker(t *testing.T) {
	suite.Run(t, new(RequestCheckerTestSuite))
}
