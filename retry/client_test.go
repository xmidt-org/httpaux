//nolint:bodyclose // none of these test use a "real" response body
package retry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/httpmock"
)

type ClientTestSuite struct {
	suite.Suite

	server    *httptest.Server
	serverURL string

	actual *http.Request

	// these are the standard expectations for anything
	// created with newRequest
	expectations []httpmock.RequestAsserter
}

var _ suite.SetupAllSuite = (*ClientTestSuite)(nil)
var _ suite.TearDownAllSuite = (*ClientTestSuite)(nil)

func (suite *ClientTestSuite) SetupSuite() {
	suite.server = httptest.NewServer(
		http.HandlerFunc(suite.handler),
	)

	suite.serverURL = suite.server.URL + "/test"

	suite.expectations = []httpmock.RequestAsserter{
		httpmock.Path("/test"),
		httpmock.Header("ClientTestSuite", "true"),
	}
}

func (suite *ClientTestSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *ClientTestSuite) handler(rw http.ResponseWriter, r *http.Request) {
	// make a safe clone of this request for assertions
	suite.actual = r.WithContext(context.Background())

	// add some known items to the response
	rw.Header().Set("ClientTestSuite", "true")
	rw.WriteHeader(299)
}

func (suite *ClientTestSuite) newRequest(method string, body io.ReadCloser) *http.Request {
	r, err := http.NewRequest(
		method,
		suite.serverURL,
		body,
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(r)
	return r
}

func (suite *ClientTestSuite) assertRequest(expectations ...httpmock.RequestAsserter) {
	a := assert.New(suite.T())

	for _, e := range suite.expectations {
		e.Assert(a, suite.actual)
	}

	for _, e := range expectations {
		e.Assert(a, suite.actual)
	}
}

func (suite *ClientTestSuite) TestDo() {
	testData := []struct {
		cfg Config
	}{
		{},
	}
}

func (suite *ClientTestSuite) TestThen() {
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
