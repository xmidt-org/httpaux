package httpaux

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func testErrorSimple(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		wrappedErr = errors.New("expected")
		err        = &Error{
			Err: wrappedErr,
		}
	)

	assert.Equal(wrappedErr, err.Unwrap())
	assert.Equal(http.StatusInternalServerError, err.StatusCode())
	assert.Empty(err.Headers())
	assert.Contains(err.Error(), "expected")

	msg, marshalErr := json.Marshal(err)
	require.NoError(marshalErr)
	assert.JSONEq(
		`{"code": 500, "cause": "expected"}`,
		string(msg),
	)
}

func testErrorNoMessage(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		wrappedErr = errors.New("expected")
		err        = &Error{
			Err:  wrappedErr,
			Code: http.StatusNotFound,
			Header: http.Header{
				"Error": {"value"},
			},
		}
	)

	assert.Equal(wrappedErr, err.Unwrap())
	assert.Equal(http.StatusNotFound, err.StatusCode())
	assert.Equal(
		http.Header{
			"Error": {"value"},
		},
		err.Headers(),
	)

	assert.Contains(err.Error(), "expected")

	msg, marshalErr := json.Marshal(err)
	require.NoError(marshalErr)
	assert.JSONEq(
		`{"code": 404, "cause": "expected"}`,
		string(msg),
	)
}

func testErrorCustomMessage(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		wrappedErr = errors.New("expected")
		err        = &Error{
			Err:     wrappedErr,
			Message: "test",
			Code:    http.StatusNotFound,
			Header: http.Header{
				"Error": {"value"},
			},
		}
	)

	assert.Equal(wrappedErr, err.Unwrap())
	assert.Equal(http.StatusNotFound, err.StatusCode())
	assert.Equal(
		http.Header{
			"Error": {"value"},
		},
		err.Headers(),
	)

	assert.Contains(err.Error(), "expected")
	assert.Contains(err.Error(), "test")

	msg, marshalErr := json.Marshal(err)
	require.NoError(marshalErr)
	assert.JSONEq(
		`{"code": 404, "message": "test", "cause": "expected"}`,
		string(msg),
	)
}

func TestError(t *testing.T) {
	t.Run("Simple", testErrorSimple)
	t.Run("NoMessage", testErrorNoMessage)
	t.Run("CustomMessage", testErrorCustomMessage)
}

func TestIsTemporary(t *testing.T) {
	assert := assert.New(t)

	assert.False(
		IsTemporary(errors.New("this isn't a temporary error")),
	)

	assert.False(
		IsTemporary(&net.DNSError{
			IsTemporary: false,
		}),
	)

	assert.True(
		IsTemporary(&net.DNSError{
			IsTemporary: true,
		}),
	)

	assert.False(
		IsTemporary(context.Canceled),
	)

	// context.DeadlineExceeded is a Temporary error
	// see: https://go.googlesource.com/go/+/go1.16/src/context/context.go#167
	assert.True(
		IsTemporary(context.DeadlineExceeded),
	)
}

type EncodeErrorSuite struct {
	suite.Suite

	response *httptest.ResponseRecorder
}

var _ suite.SetupTestSuite = (*EncodeErrorSuite)(nil)

func (suite *EncodeErrorSuite) SetupTest() {
	suite.response = httptest.NewRecorder()
}

func (suite *EncodeErrorSuite) TestSimple() {
	EncodeError(
		context.Background(),
		errors.New("expected"),
		suite.response,
	)

	result := suite.response.Result() //nolint:bodyclose
	suite.Equal(http.StatusInternalServerError, result.StatusCode)
	suite.Equal(
		http.Header{
			"Content-Type": {"application/json"},
		},
		result.Header,
	)

	body, err := ioutil.ReadAll(result.Body)
	suite.Require().NoError(err)

	suite.JSONEq(
		`{"code": 500, "cause": "expected"}`,
		string(body),
	)
}

func (suite *EncodeErrorSuite) TestCustom() {
	EncodeError(
		context.Background(),
		&Error{
			Code:    506,
			Err:     errors.New("expected"),
			Message: "here is an error",
			Header: http.Header{
				"Custom": {"true"},
			},
		},
		suite.response,
	)

	result := suite.response.Result() //nolint:bodyclose
	suite.Equal(506, result.StatusCode)
	suite.Equal(
		http.Header{
			"Content-Type": {"application/json"},
			"Custom":       {"true"},
		},
		result.Header,
	)

	body, err := ioutil.ReadAll(result.Body)
	suite.Require().NoError(err)

	suite.JSONEq(
		`{"code": 506, "message": "here is an error", "cause": "expected"}`,
		string(body),
	)
}

func TestErrorEncoder(t *testing.T) {
	suite.Run(t, new(EncodeErrorSuite))
}
