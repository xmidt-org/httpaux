package httpaux

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type HeaderTestSuite struct {
	suite.Suite
}

func (suite *HeaderTestSuite) assertHeader(expected http.Header, actual Header) bool {
	actualHeader := make(http.Header)
	actual.SetTo(actualHeader)
	return suite.Equal(expected, actualHeader)
}

func (suite *HeaderTestSuite) TestEmptyHeader() {
	h := EmptyHeader()
	suite.Zero(h.Len())

	n := h.AppendHeaders("Header1", "value1")
	suite.NotSame(n, h)
	suite.assertHeader(
		http.Header{
			"Header1": {"value1"},
		},
		n,
	)
}

func TestHeader(t *testing.T) {
	suite.Run(t, new(HeaderTestSuite))
}
