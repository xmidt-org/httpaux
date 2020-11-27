package roundtrip

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestFunc(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		expected = httptest.NewRequest("GET", "/", nil)

		called bool
		f      Func = func(actual *http.Request) (*http.Response, error) {
			called = true
			assert.Equal(expected, actual)
			return &http.Response{StatusCode: 211}, nil
		}
	)

	response, err := f.RoundTrip(expected)
	assert.True(called)
	assert.NoError(err)
	require.NotNil(response)
	assert.Equal(211, response.StatusCode)
}

func TestCloseIdlerFunc(t *testing.T) {
	var (
		assert = assert.New(t)

		called bool
		c      CloseIdlerFunc = func() {
			called = true
		}
	)

	c.CloseIdleConnections()
	assert.True(called)
}

type CloseIdleConnectionsTestSuite struct {
	suite.Suite
}

func (suite *CloseIdleConnectionsTestSuite) TestRoundTripper() {
	m := new(mockRoundTripper)
	CloseIdleConnections(m)
	m.AssertExpectations(suite.T())
}

func (suite *CloseIdleConnectionsTestSuite) TestCloseIdler() {
	m := new(mockRoundTripperCloseIdler)
	m.On("CloseIdleConnections").Once()
	CloseIdleConnections(m)
	m.AssertExpectations(suite.T())
}

func TestCloseIdleConnections(t *testing.T) {
	suite.Run(t, new(CloseIdleConnectionsTestSuite))
}

type PreserveCloseIdlerTestSuite struct {
	suite.Suite
	request  *http.Request
	response *http.Response
	err      error
}

var _ suite.BeforeTest = (*PreserveCloseIdlerTestSuite)(nil)

func (suite *PreserveCloseIdlerTestSuite) BeforeTest(_, testName string) {
	suite.request = httptest.NewRequest("GET", "/"+testName, nil)
	suite.response = &http.Response{
		StatusCode: 214,
		Header: http.Header{
			"X-Test": {testName},
		},
	}

	suite.err = fmt.Errorf("expected error: %s", testName)
}

func (suite *PreserveCloseIdlerTestSuite) TestNoCloseIdler() {
	var (
		next      = new(mockRoundTripper)
		decorator = PreserveCloseIdler(
			next,
			Func(func(r *http.Request) (*http.Response, error) {
				return next.RoundTrip(r)
			}),
		)
	)

	suite.Require().NotNil(decorator)
	next.On("RoundTrip", suite.request).Once().Return(suite.response, suite.err)

	response, err := decorator.RoundTrip(suite.request)
	suite.Equal(suite.response, response)
	suite.Equal(suite.err, err)

	_, ok := decorator.(CloseIdler)
	suite.False(ok)

	next.AssertExpectations(suite.T())
}

func (suite *PreserveCloseIdlerTestSuite) TestDecoratedCloseIdler() {
	var (
		next       = new(mockRoundTripper)
		closeIdler = new(mockRoundTripperCloseIdler)

		decorator = PreserveCloseIdler(
			next,
			Decorator{
				RoundTripper: Func(func(r *http.Request) (*http.Response, error) {
					return next.RoundTrip(r)
				}),
				CloseIdler: closeIdler,
			},
		)
	)

	suite.Require().NotNil(decorator)
	next.On("RoundTrip", suite.request).Once().Return(suite.response, suite.err)
	closeIdler.On("CloseIdleConnections").Once()

	response, err := decorator.RoundTrip(suite.request)
	suite.Equal(suite.response, response)
	suite.Equal(suite.err, err)

	suite.Require().Implements((*CloseIdler)(nil), decorator)
	decorator.(CloseIdler).CloseIdleConnections()

	next.AssertExpectations(suite.T())
	closeIdler.AssertExpectations(suite.T())
}

func (suite *PreserveCloseIdlerTestSuite) TestNextCloseIdler() {
	var (
		next      = new(mockRoundTripperCloseIdler)
		decorator = PreserveCloseIdler(
			next,
			Func(func(r *http.Request) (*http.Response, error) {
				return next.RoundTrip(r)
			}),
		)
	)

	suite.Require().NotNil(decorator)
	next.On("RoundTrip", suite.request).Once().Return(suite.response, suite.err)
	next.On("CloseIdleConnections").Once()

	response, err := decorator.RoundTrip(suite.request)
	suite.Equal(suite.response, response)
	suite.Equal(suite.err, err)

	suite.Require().Implements((*CloseIdler)(nil), decorator)
	decorator.(CloseIdler).CloseIdleConnections()

	next.AssertExpectations(suite.T())
}

func (suite *PreserveCloseIdlerTestSuite) TestNextDecorator() {
	var (
		next       = new(mockRoundTripper)
		closeIdler = new(mockRoundTripperCloseIdler)

		decorator = PreserveCloseIdler(
			Decorator{
				RoundTripper: next,
				CloseIdler:   closeIdler,
			},
			Func(func(r *http.Request) (*http.Response, error) {
				return next.RoundTrip(r)
			}),
		)
	)

	suite.Require().NotNil(decorator)
	next.On("RoundTrip", suite.request).Once().Return(suite.response, suite.err)
	closeIdler.On("CloseIdleConnections").Once()

	response, err := decorator.RoundTrip(suite.request)
	suite.Equal(suite.response, response)
	suite.Equal(suite.err, err)

	suite.Require().Implements((*CloseIdler)(nil), decorator)
	decorator.(CloseIdler).CloseIdleConnections()

	next.AssertExpectations(suite.T())
	closeIdler.AssertExpectations(suite.T())
}

func TestPreserveCloseIdler(t *testing.T) {
	suite.Run(t, new(PreserveCloseIdlerTestSuite))
}
