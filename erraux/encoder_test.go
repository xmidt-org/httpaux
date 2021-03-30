package erraux

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type EncoderTestSuite struct {
	suite.Suite
}

// result returns the status code, header, and fully read body from the recorder.
// this method also runs basic assertions.
func (suite *EncoderTestSuite) result(rw *httptest.ResponseRecorder) (statusCode int, h http.Header, body string) {
	suite.Require().NotNil(rw)
	result := rw.Result()
	result.Body.Close()

	statusCode = result.StatusCode
	suite.GreaterOrEqual(statusCode, 400)
	suite.Less(statusCode, 600)

	h = result.Header
	suite.Equal("application/json", h.Get("Content-Type"))

	b, err := ioutil.ReadAll(result.Body)
	suite.Require().NoError(err)
	body = string(b)

	return
}

// assertHeader assets that all the headers in expected are set in actual.
// each header is asserted individually.
func (suite *EncoderTestSuite) assertHeader(expected, actual http.Header) {
	for k, v := range expected {
		suite.ElementsMatchf(v, actual[k], "header [%s] values mismatch", k)
	}
}

func (suite *EncoderTestSuite) TestZeroValue() {
	response := httptest.NewRecorder()
	Encoder{}.Encode(
		context.Background(),
		errors.New("expected"),
		response,
	)

	statusCode, _, body := suite.result(response)
	suite.Equal(http.StatusInternalServerError, statusCode)
	suite.JSONEq(
		`{"code": 500, "cause": "expected"}`,
		body,
	)
}

func (suite *EncoderTestSuite) TestIs() {
	suite.Run("Simple", func() {
		simpleErr := errors.New("simple error")
		m := Encoder{}.Add(
			Is(simpleErr),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			simpleErr,
			response,
		)

		statusCode, _, body := suite.result(response)
		suite.Equal(http.StatusInternalServerError, statusCode)
		suite.JSONEq(
			`{"code": 500, "cause": "simple error"}`,
			body,
		)
	})

	suite.Run("Custom", func() {
		customErr := &Error{
			Err:     errors.New("nested"),
			Message: "here is a lovely message",
			Code:    517,
			Header: http.Header{
				"Custom": {"true"},
			},
			Fields: Fields{
				"custom": []int{123, 456},
			},
		}

		m := Encoder{}.Add(
			Is(customErr),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			customErr,
			response,
		)

		statusCode, h, body := suite.result(response)
		suite.Equal(517, statusCode)
		suite.assertHeader(
			http.Header{
				"Custom": {"true"},
			},
			h,
		)

		suite.JSONEq(
			`{"code": 517, "message": "here is a lovely message", "cause": "nested", "custom": [123, 456]}`,
			body,
		)
	})

	suite.Run("Override", func() {
		simpleErr := errors.New("simple error")
		m := Encoder{}.Add(
			Is(simpleErr).
				StatusCode(599).
				Headers("Error", "true").
				Fields("field1", "value1"),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			simpleErr,
			response,
		)

		statusCode, h, body := suite.result(response)
		suite.Equal(599, statusCode)
		suite.assertHeader(
			http.Header{"Error": {"true"}},
			h,
		)

		suite.JSONEq(
			`{"code": 599, "cause": "simple error", "field1": "value1"}`,
			body,
		)
	})

	suite.Run("Fallthrough", func() {
		simpleErr := errors.New("simple error")
		m := Encoder{}.Add(
			Is(simpleErr).
				StatusCode(512),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			errors.New("an error that is not configured"),
			response,
		)

		statusCode, _, body := suite.result(response)
		suite.Equal(http.StatusInternalServerError, statusCode)
		suite.JSONEq(
			`{"code": 500, "cause": "an error that is not configured"}`,
			body,
		)
	})
}

func (suite *EncoderTestSuite) TestAs() {
	suite.Run("NilTarget", func() {
		suite.Panics(func() {
			As(nil)
		})
	})

	suite.Run("InvalidTarget", func() {
		suite.Panics(func() {
			As(123)
		})
	})

	suite.Run("Simple", func() {
		customErr := &Error{
			Err:     errors.New("nested"),
			Message: "a message",
			Code:    567,
			Header: http.Header{
				"Custom": {"value1", "value2"},
			},
			Fields: Fields{"custom": []string{"a", "b"}},
		}

		m := Encoder{}.Add(
			As((*Error)(nil)),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			customErr,
			response,
		)

		statusCode, h, body := suite.result(response)
		suite.Equal(567, statusCode)
		suite.assertHeader(
			http.Header{
				"Custom": {"value1", "value2"},
			},
			h,
		)

		suite.JSONEq(
			`{"code": 567, "cause": "nested", "message": "a message", "custom": ["a", "b"]}`,
			body,
		)
	})
}

func TestEncoder(t *testing.T) {
	suite.Run(t, new(EncoderTestSuite))
}
