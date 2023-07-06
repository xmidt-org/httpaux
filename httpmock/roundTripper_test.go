//nolint:bodyclose,errorlint // no server responses and all errors must be unwrapped
package httpmock

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RoundTripperSuite struct {
	suite.Suite

	// server is used to test delegations
	server *httptest.Server
}

var _ suite.SetupAllSuite = (*RoundTripperSuite)(nil)
var _ suite.TearDownAllSuite = (*RoundTripperSuite)(nil)

// assertResponse verifies that the response came from the handler method
func (suite *RoundTripperSuite) assertResponse(r *http.Response) {
	suite.Require().NotNil(r)

	if suite.NotNil(r.Body) {
		defer r.Body.Close()
		b, err := io.ReadAll(r.Body)
		if suite.NoError(err) {
			suite.Equal("RoundTripperSuite", string(b))
		}
	}

	suite.Equal(
		"true",
		r.Header.Get("RoundTripperSuite"),
	)

	suite.Equal(299, r.StatusCode)
}

func (suite *RoundTripperSuite) newRequest(method, url string, body io.Reader) *http.Request {
	r, err := http.NewRequest(method, url, body)
	suite.Require().NoError(err)
	return r
}

func (suite *RoundTripperSuite) handler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("RoundTripperSuite", "true")
	rw.WriteHeader(299)
	rw.Write([]byte("RoundTripperSuite"))
}

func (suite *RoundTripperSuite) SetupSuite() {
	suite.server = httptest.NewServer(
		http.HandlerFunc(suite.handler),
	)
}

func (suite *RoundTripperSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *RoundTripperSuite) TestSuite() {
	var (
		transport   = NewRoundTripperSuite(suite)
		expected    = new(http.Response)
		expectedErr = errors.New("expected")
	)

	// everything should pass
	// a test break here is a "real" test break
	transport.OnAny().Return(expected, expectedErr).Once()
	actual, actualErr := transport.RoundTrip(new(http.Request))
	suite.True(expected == actual)
	suite.True(expectedErr == actualErr)

	transport.AssertExpectations()
}

func (suite *RoundTripperSuite) testSimpleReturn() {
	var (
		testingT    = wrapTestingT(suite.T())
		transport   = NewRoundTripper(testingT)
		expected    = new(http.Response)
		expectedErr = errors.New("expected")
	)

	transport.OnAny().Return(expected, expectedErr).Once()
	actual, actualErr := transport.RoundTrip(new(http.Request))
	suite.True(expected == actual)
	suite.True(expectedErr == actualErr)

	suite.Zero(testingT.Logs)
	suite.Zero(testingT.Errors)
	suite.Zero(testingT.Failures)
	transport.AssertExpectations()
}

func (suite *RoundTripperSuite) testSimpleResponse() {
	var (
		testingT  = wrapTestingT(suite.T())
		transport = NewRoundTripper(testingT)
		expected  = new(http.Response)
	)

	transport.OnAny().Response(expected).Once()
	actual, actualErr := transport.RoundTrip(new(http.Request))
	suite.True(expected == actual)
	suite.NoError(actualErr)

	suite.Zero(testingT.Logs)
	suite.Zero(testingT.Errors)
	suite.Zero(testingT.Failures)
	transport.AssertExpectations()
}

func (suite *RoundTripperSuite) testSimpleError() {
	var (
		testingT  = wrapTestingT(suite.T())
		transport = NewRoundTripper(testingT)
		expected  = errors.New("expected")
	)

	transport.OnAny().Error(expected).Once()
	response, actual := transport.RoundTrip(new(http.Request))
	suite.Nil(response)
	suite.Same(expected, actual)

	suite.Zero(testingT.Logs)
	suite.Zero(testingT.Errors)
	suite.Zero(testingT.Failures)
	transport.AssertExpectations()
}

func (suite *RoundTripperSuite) TestSimple() {
	suite.Run("Return", suite.testSimpleReturn)
	suite.Run("Response", suite.testSimpleResponse)
	suite.Run("Error", suite.testSimpleError)
}

func (suite *RoundTripperSuite) TestMockRequestAssertions() {
	suite.Run("Pass", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/test",
				},
				Header: http.Header{
					"Single-Value": {"value1"},
					"Multi-Value":  {"value2", "value3"},
				},
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		transport.AssertRequest(
			Methods("GET", "POST"),
			Path("/test"),
			Header("Single-Value", "value1"),
			Header("Multi-Value", "value2", "value3"),
		).OnAny().Return(expected, expectedErr)
		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)

		suite.Zero(testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})

	suite.Run("Fail", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "POST",
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		transport.AssertRequest(
			Methods("PATCH"),
		).OnAny().Return(expected, expectedErr)
		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)

		suite.Zero(testingT.Logs)
		suite.Equal(1, testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})
}

func (suite *RoundTripperSuite) TestCallRequestAssertions() {
	suite.Run("Pass", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/test",
				},
				Header: http.Header{
					"Single-Value": {"value1"},
					"Multi-Value":  {"value2", "value3"},
				},
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		// assertion is defined on the Call, not on the mock
		transport.OnAny().
			AssertRequest(
				Methods("GET", "POST"),
				Path("/test"),
				Header("Single-Value", "value1"),
				Header("Multi-Value", "value2", "value3"),
			).Return(expected, expectedErr)
		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)

		suite.Zero(testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})

	suite.Run("Fail", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "POST",
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		// assertion is defined on the Call, not on the mock
		transport.OnAny().
			AssertRequest(
				Methods("PATCH"),
			).Return(expected, expectedErr)
		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)

		suite.Zero(testingT.Logs)
		suite.Equal(1, testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})
}

func (suite *RoundTripperSuite) TestOnRequest() {
	suite.Run("Pass", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = new(http.Request)

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		transport.OnRequest(request).Return(expected, expectedErr)
		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)

		suite.Zero(testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})

	suite.Run("Fail", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = new(http.Request)

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		transport.OnRequest(request).Return(expected, expectedErr)

		suite.Panics(func() {
			transport.RoundTrip(new(http.Request)) // different instance
		})

		suite.Zero(testingT.Logs)
		suite.Equal(1, testingT.Errors)
		suite.Equal(1, testingT.Failures)
	})
}

func (suite *RoundTripperSuite) TestOnMatchAll() {
	suite.Run("Pass", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/test",
				},
				Header: http.Header{
					"Test": {"true"},
				},
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		transport.OnMatchAll(
			Methods("POST"),
			Path("/test"),
			Header("Test", "true"),
		).Return(expected, expectedErr).Once()
		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)

		suite.Zero(testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})

	suite.Run("Fail", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/test",
				},
				Header: http.Header{
					"Test": {"true"},
				},
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		transport.OnMatchAll(
			Path("/test"),
			Methods("POST"),
			Header("Test", "true"),
		).Return(expected, expectedErr).Once()

		suite.Panics(func() {
			transport.RoundTrip(request)
		})

		suite.Zero(testingT.Logs)
		suite.Equal(1, testingT.Errors)
		suite.Equal(1, testingT.Failures)
	})
}

func (suite *RoundTripperSuite) TestOnMatchAny() {
	suite.Run("Pass", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "POST",
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		transport.OnMatchAny(
			Methods("POST"),
			Path("/test"),
			Header("Test", "true"),
		).Return(expected, expectedErr).Once()
		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)

		suite.Zero(testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})

	suite.Run("Fail", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "GET",
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
		)

		transport.OnMatchAny(
			Path("/test"),
			Methods("POST"),
			Header("Test", "true"),
		).Return(expected, expectedErr).Once()

		suite.Panics(func() {
			transport.RoundTrip(request)
		})

		suite.Zero(testingT.Logs)
		suite.Equal(1, testingT.Errors)
		suite.Equal(1, testingT.Failures)
	})
}

func (suite *RoundTripperSuite) TestCustomRun() {
	suite.Run("NoAssertions", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "GET",
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
			runCalled   bool
		)

		transport.OnAny().Run(func(args mock.Arguments) {
			runCalled = true
		}).Return(expected, expectedErr).Once()

		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)
		suite.True(runCalled)

		suite.Zero(testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})

	suite.Run("AssertionsPass", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "GET",
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
			runCalled   bool
		)

		transport.OnAny().AssertRequest(
			Methods("GET"),
		).Run(func(args mock.Arguments) {
			runCalled = true
		}).Return(expected, expectedErr).Once()

		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)
		suite.True(runCalled)

		suite.Zero(testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})

	suite.Run("AssertionsFail", func() {
		var (
			testingT  = wrapTestingT(suite.T())
			transport = NewRoundTripper(testingT)
			request   = &http.Request{
				Method: "GET",
			}

			expected    = new(http.Response)
			expectedErr = errors.New("expected")
			runCalled   bool
		)

		// if assertions fail, the run function should still execute
		transport.OnAny().AssertRequest(
			Methods("POST"),
		).Run(func(args mock.Arguments) {
			runCalled = true
		}).Return(expected, expectedErr).Once()

		actual, actualErr := transport.RoundTrip(request)
		suite.True(expected == actual)
		suite.True(expectedErr == actualErr)
		suite.True(runCalled)

		suite.Zero(testingT.Logs)
		suite.Equal(1, testingT.Errors)
		suite.Zero(testingT.Failures)
		transport.AssertExpectations()
	})
}

func (suite *RoundTripperSuite) TestNext() {
	suite.Run("Nil", func() {
		var (
			testingT = wrapTestingT(suite.T())
			rt       = NewRoundTripper(testingT)
		)

		rt.Next(nil) // this should be valid, as it will use http.DefaultTransport
		rt.OnAny()   // no return needed, since we set a Next

		response, err := rt.RoundTrip(suite.newRequest("GET", suite.server.URL+"/test", nil))
		suite.NoError(err)
		suite.assertResponse(response)

		rt.AssertExpectations()
		suite.Equal(1, testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
	})

	suite.Run("Custom", func() {
		var (
			testingT = wrapTestingT(suite.T())
			rt       = NewRoundTripper(testingT)
		)

		rt.Next(new(http.Transport))
		rt.OnAny() // no return needed, since we set a Next

		response, err := rt.RoundTrip(suite.newRequest("GET", suite.server.URL+"/test", nil))
		suite.NoError(err)
		suite.assertResponse(response)

		rt.AssertExpectations()
		suite.Equal(1, testingT.Logs)
		suite.Zero(testingT.Errors)
		suite.Zero(testingT.Failures)
	})
}

func TestRoundTripper(t *testing.T) {
	suite.Run(t, new(RoundTripperSuite))
}

func TestCloseIdler(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		m       = &CloseIdler{
			RoundTripper: NewRoundTripper(t),
		}

		c = http.Client{
			Transport: m,
		}

		expected = &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   "localhost",
				Path:   "/testCloseIdler",
			},
		}
	)

	m.OnMatchAll(
		Path("/testCloseIdler"),
	).Return(&http.Response{StatusCode: 288}, nil).Once()
	m.OnCloseIdleConnections().Once()

	r, err := c.Do(expected)
	assert.NoError(err)
	require.NotNil(r)
	defer r.Body.Close()
	assert.Equal(288, r.StatusCode)
	c.CloseIdleConnections()

	m.AssertExpectations()
}
