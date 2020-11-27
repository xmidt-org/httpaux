package roundtrip

import (
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

func TestConstructor(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		expected = &http.Transport{
			MaxResponseHeaderBytes: 1234,
		}

		called bool
		c      Constructor = func(actual http.RoundTripper) http.RoundTripper {
			called = true
			assert.Equal(expected, actual)
			return actual
		}
	)

	decorated := c.Then(expected)
	assert.True(called)
	require.NotNil(decorated)
	assert.Equal(expected, decorated)
}

type CloseIdleConnectionsTestSuite struct {
	suite.Suite
	roundTripper *mockRoundTripper
	closeIdler   *mockRoundTripperCloseIdler
}

var _ suite.SetupTestSuite = (*CloseIdleConnectionsTestSuite)(nil)
var _ suite.TearDownTestSuite = (*CloseIdleConnectionsTestSuite)(nil)

func (suite *CloseIdleConnectionsTestSuite) SetupTest() {
	suite.roundTripper = new(mockRoundTripper)
	suite.closeIdler = new(mockRoundTripperCloseIdler)
}

func (suite *CloseIdleConnectionsTestSuite) TearDownTest() {
	suite.roundTripper.AssertExpectations(suite.T())
	suite.closeIdler.AssertExpectations(suite.T())
}

func (suite *CloseIdleConnectionsTestSuite) TestRoundTripper() {
	CloseIdleConnections(suite.roundTripper)
}

func (suite *CloseIdleConnectionsTestSuite) TestCloseIdler() {
	suite.closeIdler.On("CloseIdleConnections").Once()
	CloseIdleConnections(suite.closeIdler)
}

func TestCloseIdleConnections(t *testing.T) {
	suite.Run(t, new(CloseIdleConnectionsTestSuite))
}
