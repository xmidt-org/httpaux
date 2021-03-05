package httpmock

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

type BodyReadCloserTestSuite struct {
	suite.Suite
}

func (suite *BodyReadCloserTestSuite) testBodyReadCloser(body *BodyReadCloser, expected []byte) {
	suite.Require().NotNil(body)

	actual, err := ioutil.ReadAll(body)
	suite.Equal(expected, actual)
	suite.NoError(err)

	_, err = body.Read(make([]byte, 2))
	suite.Equal(io.EOF, err)

	suite.False(body.Closed())
	suite.False(Closed(body))
	suite.NoError(body.Close())
	suite.Equal(ErrBodyClosed, body.Close())

	_, err = body.Read(make([]byte, 2))
	suite.Equal(ErrBodyClosed, err)
	suite.True(body.Closed())
	suite.True(Closed(body))
}

func (suite *BodyReadCloserTestSuite) TestEmptyBody() {
	body := EmptyBody()
	suite.testBodyReadCloser(body, []byte{})
}

func (suite *BodyReadCloserTestSuite) TestBodyBytes() {
	const bodyContents = "some lovely content here"
	body := BodyBytes([]byte(bodyContents))
	suite.testBodyReadCloser(body, []byte(bodyContents))
}

func (suite *BodyReadCloserTestSuite) TestBodyString() {
	const bodyContents = "some lovely content here"
	body := BodyString(bodyContents)
	suite.testBodyReadCloser(body, []byte(bodyContents))
}

func (suite *BodyReadCloserTestSuite) TestBodyf() {
	const bodyContents = "some lovely content here"
	body := Bodyf("Format string: %d", 123)
	suite.testBodyReadCloser(body, []byte(fmt.Sprintf("Format string: %d", 123)))
}

func (suite *BodyReadCloserTestSuite) TestNopCloser() {
	body := ioutil.NopCloser(
		bytes.NewBufferString("this body doesn't implement Closeable"),
	)

	suite.Panics(func() {
		Closed(body)
	})
}

func TestBodyReadCloser(t *testing.T) {
	suite.Run(t, new(BodyReadCloserTestSuite))
}
