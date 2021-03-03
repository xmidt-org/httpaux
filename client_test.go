package httpaux

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testBody struct {
	bytes.Buffer
	closed bool
}

func (tb *testBody) Close() error {
	tb.closed = true
	return nil
}

type CleanupTestSuite struct {
	suite.Suite
}

func (suite *CleanupTestSuite) TestNil() {
	Cleanup(nil)
}

func (suite *CleanupTestSuite) TestNilBody() {
	Cleanup(new(http.Response))
}

func (suite *CleanupTestSuite) TestClosed() {
	tb := new(testBody)
	tb.Buffer.WriteString("test")
	Cleanup(&http.Response{
		Body: tb,
	})

	suite.Zero(tb.Len())
}

func TestCleanup(t *testing.T) {
	suite.Run(t, new(CleanupTestSuite))
}
