// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package erraux

import (
	"context"
	"errors"
	"io"
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

	b, err := io.ReadAll(result.Body)
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

	suite.Run("NoBody", func() {
		simpleErr := errors.New("this should not appear")
		m := Encoder{}.Body(false).Add(
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
		suite.Empty(body)
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

	suite.Run("CustomRule", func() {
		simpleErr := errors.New("simple error")
		m := Encoder{}.Add(
			Is(simpleErr).
				StatusCode(599).
				Cause("a custom cause").
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
			`{"code": 599, "cause": "a custom cause", "field1": "value1"}`,
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

// TestInterface is used in the As tests
type TestInterface interface {
	DoIt()
}

// testError implements TestInterface
type testError struct{}

func (te *testError) Error() string { return "test error" }
func (te *testError) DoIt()         {}

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

	suite.Run("NoBody", func() {
		customErr := &Error{
			Err:     errors.New("nested"),
			Message: "a message",
			Code:    567,
			Header: http.Header{
				"Custom": {"value1", "value2"},
			},
			Fields: Fields{"custom": []string{"a", "b"}},
		}

		m := Encoder{}.Body(false).
			Add(As((*Error)(nil)))
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

		suite.Empty(body)
	})

	suite.Run("CustomRule", func() {
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
			As((*Error)(nil)).
				StatusCode(598).
				Cause("a custom cause").
				Headers("Additional", "true").
				Fields("additional", 45.9),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			customErr,
			response,
		)

		statusCode, h, body := suite.result(response)
		suite.Equal(598, statusCode)
		suite.assertHeader(
			http.Header{
				"Custom":     {"value1", "value2"},
				"Additional": {"true"},
			},
			h,
		)

		suite.JSONEq(
			`{"code": 598, "cause": "a custom cause", "message": "a message", "custom": ["a", "b"], "additional": 45.9}`,
			body,
		)
	})

	suite.Run("Interface", func() {
		m := Encoder{}.Add(
			As((*TestInterface)(nil)),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			&testError{},
			response,
		)

		statusCode, _, body := suite.result(response)
		suite.Equal(500, statusCode)
		suite.JSONEq(
			`{"code": 500, "cause": "test error"}`,
			body,
		)
	})

	suite.Run("InterfaceWithCustomRule", func() {
		m := Encoder{}.Add(
			As((*TestInterface)(nil)).
				StatusCode(512).
				Headers("Custom", "true").
				Fields("custom", 123),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			&testError{},
			response,
		)

		statusCode, h, body := suite.result(response)
		suite.Equal(512, statusCode)
		suite.assertHeader(
			http.Header{
				"Custom": {"true"},
			},
			h,
		)

		suite.JSONEq(
			`{"code": 512, "cause": "test error", "custom": 123}`,
			body,
		)
	})

	suite.Run("Fallthrough", func() {
		m := Encoder{}.Add(
			As((*TestInterface)(nil)),
		)

		response := httptest.NewRecorder()
		m.Encode(
			context.Background(),
			errors.New("this error is not defined"),
			response,
		)

		statusCode, _, body := suite.result(response)
		suite.Equal(500, statusCode)
		suite.JSONEq(
			`{"code": 500, "cause": "this error is not defined"}`,
			body,
		)
	})
}

func (suite *EncoderTestSuite) TestFallthroughWithCustom() {
	response := httptest.NewRecorder()
	Encoder{}.Encode(
		context.Background(),
		&Error{
			Err:    errors.New("cause"),
			Code:   592,
			Header: http.Header{"Custom": {"true"}},
			Fields: Fields{"foo": "bar"},
		},
		response,
	)

	statusCode, h, body := suite.result(response)
	suite.Equal(592, statusCode)
	suite.assertHeader(
		http.Header{
			"Custom": {"true"},
		},
		h,
	)

	suite.JSONEq(
		`{"code": 592, "cause": "cause", "foo": "bar"}`,
		body,
	)
}

func TestEncoder(t *testing.T) {
	suite.Run(t, new(EncoderTestSuite))
}
