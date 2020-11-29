package observe

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux"
)

type ThenTestSuite struct {
	suite.Suite
	next     http.Handler
	response *httptest.ResponseRecorder
}

func (suite *ThenTestSuite) SetupSuite() {
	suite.next = httpaux.ConstantHandler{
		StatusCode: 217,
	}
}

func (suite *ThenTestSuite) SetupTest() {
	suite.response = httptest.NewRecorder()
}

func (suite *ThenTestSuite) assertNextIsDecorated(decorated http.Handler) {
	suite.Require().NotNil(decorated)
	decorated.ServeHTTP(suite.response, httptest.NewRequest("GET", "/test", nil))
	suite.Equal(217, suite.response.Code)
}

func (suite *ThenTestSuite) TestDecoration() {
	suite.assertNextIsDecorated(
		Then(suite.next),
	)
}

func (suite *ThenTestSuite) TestIdempotency() {
	first := Then(suite.next)
	second := Then(first)

	suite.Equal(second, first)
	suite.assertNextIsDecorated(second)
}

func TestThen(t *testing.T) {
	suite.Run(t, new(ThenTestSuite))
}
