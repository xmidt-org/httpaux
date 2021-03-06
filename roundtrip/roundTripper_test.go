package roundtrip

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/httpmock"
)

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
	m := httpmock.NewRoundTripperSuite(suite)
	CloseIdleConnections(m)
	m.AssertExpectations()
}

func (suite *CloseIdleConnectionsTestSuite) TestCloseIdler() {
	m := &httpmock.CloseIdler{
		RoundTripper: httpmock.NewRoundTripperSuite(suite),
	}

	m.OnCloseIdleConnections().Once()
	CloseIdleConnections(m)
	m.AssertExpectations()
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
		Body: httpmock.EmptyBody(),
	}

	suite.err = fmt.Errorf("expected error: %s", testName)
}

func (suite *PreserveCloseIdlerTestSuite) TestNoCloseIdler() {
	var (
		next      = httpmock.NewRoundTripperSuite(suite)
		decorator = PreserveCloseIdler(
			next,
			Func(func(r *http.Request) (*http.Response, error) {
				return next.RoundTrip(r)
			}),
		)
	)

	suite.Require().NotNil(decorator)
	next.OnRequest(suite.request).Once().Return(suite.response, suite.err)

	response, err := decorator.RoundTrip(suite.request)
	suite.Require().NotNil(response)
	defer response.Body.Close()
	suite.Equal(suite.response, response)
	suite.Equal(suite.err, err)

	_, ok := decorator.(CloseIdler)
	suite.False(ok)

	next.AssertExpectations()
}

func (suite *PreserveCloseIdlerTestSuite) TestDecoratedCloseIdler() {
	var (
		next       = httpmock.NewRoundTripperSuite(suite)
		closeIdler = &httpmock.CloseIdler{
			RoundTripper: httpmock.NewRoundTripperSuite(suite),
		}

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
	next.OnRequest(suite.request).Return(suite.response, suite.err).Once()
	closeIdler.OnCloseIdleConnections().Once()

	response, err := decorator.RoundTrip(suite.request)
	suite.Require().NotNil(response)
	defer response.Body.Close()
	suite.Equal(suite.response, response)
	suite.Equal(suite.err, err)

	suite.Require().Implements((*CloseIdler)(nil), decorator)
	decorator.(CloseIdler).CloseIdleConnections()

	next.AssertExpectations()
	closeIdler.AssertExpectations()
}

func (suite *PreserveCloseIdlerTestSuite) TestNextCloseIdler() {
	var (
		next = &httpmock.CloseIdler{
			RoundTripper: httpmock.NewRoundTripperSuite(suite),
		}

		decorator = PreserveCloseIdler(
			next,
			Func(func(r *http.Request) (*http.Response, error) {
				return next.RoundTrip(r)
			}),
		)
	)

	suite.Require().NotNil(decorator)
	next.OnRequest(suite.request).Once().Return(suite.response, suite.err)
	next.OnCloseIdleConnections().Once()

	response, err := decorator.RoundTrip(suite.request)
	suite.Require().NotNil(response)
	defer response.Body.Close()
	suite.Equal(suite.response, response)
	suite.Equal(suite.err, err)

	suite.Require().Implements((*CloseIdler)(nil), decorator)
	decorator.(CloseIdler).CloseIdleConnections()

	next.AssertExpectations()
}

func (suite *PreserveCloseIdlerTestSuite) TestNextDecorator() {
	var (
		next       = httpmock.NewRoundTripperSuite(suite)
		closeIdler = &httpmock.CloseIdler{
			RoundTripper: httpmock.NewRoundTripperSuite(suite),
		}

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
	next.OnRequest(suite.request).Once().Return(suite.response, suite.err)
	closeIdler.OnCloseIdleConnections().Once()

	response, err := decorator.RoundTrip(suite.request)
	suite.Require().NotNil(response)
	defer response.Body.Close()
	suite.Equal(suite.response, response)
	suite.Equal(suite.err, err)

	suite.Require().Implements((*CloseIdler)(nil), decorator)
	decorator.(CloseIdler).CloseIdleConnections()

	next.AssertExpectations()
	closeIdler.AssertExpectations()
}

func TestPreserveCloseIdler(t *testing.T) {
	suite.Run(t, new(PreserveCloseIdlerTestSuite))
}
