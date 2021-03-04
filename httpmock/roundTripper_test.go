package httpmock

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RoundTripperTestSuite struct {
	suite.Suite
}

func (suite *RoundTripperTestSuite) TestSuite() {
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

func (suite *RoundTripperTestSuite) TestSimple() {
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

func (suite *RoundTripperTestSuite) TestMockRequestAssertions() {
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

func (suite *RoundTripperTestSuite) TestCallRequestAssertions() {
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

func (suite *RoundTripperTestSuite) TestOnRequest() {
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

func TestRoundTripper(t *testing.T) {
	suite.Run(t, new(RoundTripperTestSuite))
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
		func(r *http.Request) bool {
			return r.URL.Path == "/testCloseIdler"
		},
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
