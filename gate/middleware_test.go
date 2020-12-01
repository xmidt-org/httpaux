package gate

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/roundtrip"
)

type ServerCustomTestSuite struct {
	suite.Suite
	next   http.Handler
	closed http.Handler

	gate     Interface
	response *httptest.ResponseRecorder
	request  *http.Request
}

var _ suite.SetupAllSuite = (*ServerCustomTestSuite)(nil)
var _ suite.SetupTestSuite = (*ServerCustomTestSuite)(nil)

func (suite *ServerCustomTestSuite) SetupSuite() {
	suite.next = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(299)
	})

	suite.closed = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(599)
	})
}

func (suite *ServerCustomTestSuite) SetupTest() {
	suite.gate = New(Config{
		Name: "testServer",
	})

	suite.response = httptest.NewRecorder()
	suite.request = httptest.NewRequest("GET", "/", nil)
}

func (suite *ServerCustomTestSuite) TestDefaultOpen() {
	handler := Server{Gate: suite.gate}.Then(suite.next)
	suite.Require().NotNil(handler)

	handler.ServeHTTP(suite.response, suite.request)
	suite.Equal(299, suite.response.Code)
}

func (suite *ServerCustomTestSuite) TestDefaultClosed() {
	suite.Require().True(suite.gate.Close())
	handler := Server{Gate: suite.gate}.Then(suite.next)
	suite.Require().NotNil(handler)

	handler.ServeHTTP(suite.response, suite.request)
	suite.Equal(http.StatusServiceUnavailable, suite.response.Code)
}

func (suite *ServerCustomTestSuite) TestCustomOpen() {
	handler := Server{
		Closed: suite.closed,
		Gate:   suite.gate,
	}.Then(suite.next)
	suite.Require().NotNil(handler)

	handler.ServeHTTP(suite.response, suite.request)
	suite.Equal(299, suite.response.Code)
}

func (suite *ServerCustomTestSuite) TestCustomClosed() {
	suite.Require().True(suite.gate.Close())
	handler := Server{
		Closed: suite.closed,
		Gate:   suite.gate,
	}.Then(suite.next)
	suite.Require().NotNil(handler)

	handler.ServeHTTP(suite.response, suite.request)
	suite.Equal(599, suite.response.Code)
}

func TestServerCustom(t *testing.T) {
	suite.Run(t, new(ServerCustomTestSuite))
}

type ClientCustomTestSuite struct {
	suite.Suite
	server *httptest.Server

	closed          http.RoundTripper
	customClosedErr error

	gate    Interface
	request *http.Request
}

var _ suite.SetupAllSuite = (*ClientCustomTestSuite)(nil)
var _ suite.SetupTestSuite = (*ClientCustomTestSuite)(nil)
var _ suite.TearDownAllSuite = (*ClientCustomTestSuite)(nil)

func (suite *ClientCustomTestSuite) SetupSuite() {
	suite.customClosedErr = errors.New("expected closed error")
	suite.closed = roundtrip.Func(func(*http.Request) (*http.Response, error) {
		return nil, suite.customClosedErr
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/test", suite.testServerHandle)
	suite.server = httptest.NewServer(mux)
}

func (suite *ClientCustomTestSuite) SetupTest() {
	suite.gate = New(Config{
		Name: "testClient",
	})

	var err error
	suite.request, err = http.NewRequest("GET", suite.server.URL+"/test", nil)
	suite.Require().NoError(err)
}

func (suite *ClientCustomTestSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *ClientCustomTestSuite) testServerHandle(response http.ResponseWriter, _ *http.Request) {
	response.WriteHeader(277)
}

func (suite *ClientCustomTestSuite) checkRoundTripper(rt http.RoundTripper) (*http.Response, error) {
	suite.Require().NotNil(rt)

	type checkCloseIdler interface {
		CloseIdleConnections()
	}

	suite.Require().Implements((*checkCloseIdler)(nil), rt)
	suite.NotPanics(func() {
		rt.(checkCloseIdler).CloseIdleConnections()
	})

	request, err := http.NewRequest("GET", suite.server.URL+"/test", nil)
	suite.Require().NoError(err)

	response, err := rt.RoundTrip(request)
	if response != nil {
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	}

	return response, err
}

func (suite *ClientCustomTestSuite) TestDefaultOpen() {
	suite.Run("WithNext", func() {
		rt := Client{Gate: suite.gate}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		suite.NoError(err)
		suite.Require().NotNil(response)
		suite.Equal(277, response.StatusCode)
	})

	suite.Run("NilNext", func() {
		rt := Client{Gate: suite.gate}.ThenRoundTrip(nil)
		response, err := suite.checkRoundTripper(rt)

		suite.NoError(err)
		suite.Require().NotNil(response)
		suite.Equal(277, response.StatusCode)
	})
}

func (suite *ClientCustomTestSuite) TestDefaultClosed() {
	suite.Require().True(suite.gate.Close())

	suite.Run("WithNext", func() {
		rt := Client{Gate: suite.gate}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		suite.Nil(response)
		suite.IsType((*ClosedError)(nil), err)
	})

	suite.Run("NilNext", func() {
		rt := Client{Gate: suite.gate}.ThenRoundTrip(nil)
		response, err := suite.checkRoundTripper(rt)

		suite.Nil(response)
		suite.IsType((*ClosedError)(nil), err)
	})
}

func (suite *ClientCustomTestSuite) TestCustomOpen() {
	suite.Run("WithNext", func() {
		rt := Client{
			Closed: suite.closed,
			Gate:   suite.gate,
		}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		suite.NoError(err)
		suite.Require().NotNil(response)
		suite.Equal(277, response.StatusCode)
	})

	suite.Run("NilNext", func() {
		rt := Client{
			Closed: suite.closed,
			Gate:   suite.gate,
		}.ThenRoundTrip(nil)
		response, err := suite.checkRoundTripper(rt)

		suite.NoError(err)
		suite.Require().NotNil(response)
		suite.Equal(277, response.StatusCode)
	})
}

func (suite *ClientCustomTestSuite) TestCustomClosed() {
	suite.Require().True(suite.gate.Close())

	suite.Run("WithNext", func() {
		rt := Client{
			Closed: suite.closed,
			Gate:   suite.gate,
		}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		suite.Nil(response)
		suite.Equal(suite.customClosedErr, err)
	})

	suite.Run("NilNext", func() {
		rt := Client{
			Closed: suite.closed,
			Gate:   suite.gate,
		}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		suite.Nil(response)
		suite.Equal(suite.customClosedErr, err)
	})
}

func TestClientCustom(t *testing.T) {
	suite.Run(t, new(ClientCustomTestSuite))
}
