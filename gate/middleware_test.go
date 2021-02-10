package gate

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/roundtrip"
)

type ServerTestSuite struct {
	suite.Suite
	next   http.Handler
	closed http.Handler

	gate     Interface
	response *httptest.ResponseRecorder
	request  *http.Request
}

var _ suite.SetupAllSuite = (*ServerTestSuite)(nil)
var _ suite.SetupTestSuite = (*ServerTestSuite)(nil)

func (suite *ServerTestSuite) SetupSuite() {
	suite.next = httpaux.ConstantHandler{
		StatusCode: 299,
	}

	suite.closed = httpaux.ConstantHandler{
		StatusCode: 599,
	}
}

func (suite *ServerTestSuite) SetupTest() {
	suite.gate = New(Config{
		Name: "testServer",
	})

	suite.response = httptest.NewRecorder()
	suite.request = httptest.NewRequest("GET", "/", nil)
}

func (suite *ServerTestSuite) TestNilGate() {
	handler := Server{}.Then(suite.next)
	suite.Require().NotNil(handler)
	suite.Equal(suite.next, handler)
}

func (suite *ServerTestSuite) TestDefaultOpen() {
	handler := Server{Gate: suite.gate}.Then(suite.next)
	suite.Require().NotNil(handler)

	handler.ServeHTTP(suite.response, suite.request)
	suite.Equal(299, suite.response.Code)
}

func (suite *ServerTestSuite) TestDefaultClosed() {
	suite.Require().True(suite.gate.Close())
	handler := Server{Gate: suite.gate}.Then(suite.next)
	suite.Require().NotNil(handler)

	handler.ServeHTTP(suite.response, suite.request)
	suite.Equal(http.StatusServiceUnavailable, suite.response.Code)
}

func (suite *ServerTestSuite) TestCustomOpen() {
	handler := Server{
		Closed: suite.closed,
		Gate:   suite.gate,
	}.Then(suite.next)
	suite.Require().NotNil(handler)

	handler.ServeHTTP(suite.response, suite.request)
	suite.Equal(299, suite.response.Code)
}

func (suite *ServerTestSuite) TestCustomClosed() {
	suite.Require().True(suite.gate.Close())
	handler := Server{
		Closed: suite.closed,
		Gate:   suite.gate,
	}.Then(suite.next)
	suite.Require().NotNil(handler)

	handler.ServeHTTP(suite.response, suite.request)
	suite.Equal(599, suite.response.Code)
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
	server *httptest.Server

	closed          http.RoundTripper
	customClosedErr error

	gate    Interface
	request *http.Request
}

var _ suite.SetupAllSuite = (*ClientTestSuite)(nil)
var _ suite.SetupTestSuite = (*ClientTestSuite)(nil)
var _ suite.TearDownAllSuite = (*ClientTestSuite)(nil)

func (suite *ClientTestSuite) SetupSuite() {
	suite.customClosedErr = errors.New("expected closed error")
	suite.closed = roundtrip.Func(func(*http.Request) (*http.Response, error) {
		return nil, suite.customClosedErr
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/test", suite.testServerHandle)
	suite.server = httptest.NewServer(mux)
}

func (suite *ClientTestSuite) SetupTest() {
	suite.gate = New(Config{
		Name: "testClient",
	})

	var err error
	suite.request, err = http.NewRequest("GET", suite.server.URL+"/test", nil)
	suite.Require().NoError(err)
}

func (suite *ClientTestSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *ClientTestSuite) testServerHandle(response http.ResponseWriter, _ *http.Request) {
	response.WriteHeader(277)
}

func (suite *ClientTestSuite) checkRoundTripper(rt http.RoundTripper) (*http.Response, error) {
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

	return rt.RoundTrip(request)
}

func (suite *ClientTestSuite) TestNilGate() {
	// check that no decoration happened
	next := new(http.Transport)
	suite.Equal(
		next,
		Client{}.ThenRoundTrip(next),
	)
}

func (suite *ClientTestSuite) TestDefaultOpen() {
	suite.Run("WithNext", func() {
		rt := Client{Gate: suite.gate}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		suite.NoError(err)
		suite.Require().NotNil(response)
		suite.Equal(277, response.StatusCode)
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	})

	suite.Run("NilNext", func() {
		rt := Client{Gate: suite.gate}.ThenRoundTrip(nil)
		response, err := suite.checkRoundTripper(rt)

		suite.NoError(err)
		suite.Require().NotNil(response)
		suite.Equal(277, response.StatusCode)
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	})
}

func (suite *ClientTestSuite) TestDefaultClosed() {
	suite.Require().True(suite.gate.Close())

	suite.Run("WithNext", func() {
		rt := Client{Gate: suite.gate}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		if !suite.Nil(response) {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
		}

		suite.IsType((*ClosedError)(nil), err)
	})

	suite.Run("NilNext", func() {
		rt := Client{Gate: suite.gate}.ThenRoundTrip(nil)
		response, err := suite.checkRoundTripper(rt)

		if !suite.Nil(response) {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
		}

		suite.IsType((*ClosedError)(nil), err)
	})
}

func (suite *ClientTestSuite) TestCustomOpen() {
	suite.Run("WithNext", func() {
		rt := Client{
			Closed: suite.closed,
			Gate:   suite.gate,
		}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		suite.NoError(err)
		suite.Require().NotNil(response)
		suite.Equal(277, response.StatusCode)
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
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
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	})
}

func (suite *ClientTestSuite) TestCustomClosed() {
	suite.Require().True(suite.gate.Close())

	suite.Run("WithNext", func() {
		rt := Client{
			Closed: suite.closed,
			Gate:   suite.gate,
		}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		if !suite.Nil(response) {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
		}

		suite.Equal(suite.customClosedErr, err)
	})

	suite.Run("NilNext", func() {
		rt := Client{
			Closed: suite.closed,
			Gate:   suite.gate,
		}.ThenRoundTrip(new(http.Transport))
		response, err := suite.checkRoundTripper(rt)

		if !suite.Nil(response) {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
		}

		suite.Equal(suite.customClosedErr, err)
	})
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
