// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package recovery

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux"
)

type enrichedPanicValue struct {
	message    string
	statusCode int
	headers    http.Header
}

func (epv enrichedPanicValue) String() string {
	return epv.message
}

func (epv enrichedPanicValue) StatusCode() int {
	return epv.statusCode
}

func (epv enrichedPanicValue) Headers() http.Header {
	return epv.headers
}

type MiddlewareSuite struct {
	suite.Suite
}

func (suite *MiddlewareSuite) newTestRequest() *http.Request {
	return httptest.NewRequest("GET", "/", nil)
}

// panicHandler creates a handler that panics with an expected value.
func (suite *MiddlewareSuite) panicHandler(v interface{}) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		// check that the request was unmolested
		suite.Equal("GET", request.Method)
		suite.Equal("/", request.URL.String())

		panic(v)
	})
}

func (suite *MiddlewareSuite) serveDecoratedHTTP(decorated http.Handler) *httptest.ResponseRecorder {
	suite.Require().NotNil(decorated)
	response := httptest.NewRecorder()
	response.Body = new(bytes.Buffer)
	decorated.ServeHTTP(response, suite.newTestRequest())

	return response
}

func (suite *MiddlewareSuite) TestNoPanic() {
	var (
		handler http.HandlerFunc = func(response http.ResponseWriter, request *http.Request) {
			// check that the request was unmolested
			suite.Equal("GET", request.Method)
			suite.Equal("/", request.URL.String())

			response.WriteHeader(255)
		}

		decorated = Middleware()(
			handler,
		)

		response = suite.serveDecoratedHTTP(decorated)
	)

	suite.Equal(255, response.Code)
	suite.Zero(response.Body.Len())
}

func (suite *MiddlewareSuite) TestDefault() {
	var (
		expected  = "expected panic value"
		decorated = Middleware()(
			suite.panicHandler(expected),
		)

		response = suite.serveDecoratedHTTP(decorated)
	)

	suite.Equal(http.StatusInternalServerError, response.Code)
	suite.Contains(response.Body.String(), expected)
}

func (suite *MiddlewareSuite) TestFullyCustomized() {
	var (
		expected = "expected panic value from a fully customized middleware"

		customRecoverBody             = "custom recover body"
		body              RecoverBody = func(w io.Writer, r interface{}, stack []byte) {
			suite.Equal(expected, r)
			suite.NotEmpty(stack)
			w.Write([]byte(customRecoverBody))
		}

		onRecover1Called           = false
		onRecover1       OnRecover = func(r interface{}, stack []byte) {
			onRecover1Called = true
			suite.Equal(expected, r)
			suite.NotEmpty(stack)
		}

		onRecover2Called           = false
		onRecover2       OnRecover = func(r interface{}, stack []byte) {
			onRecover2Called = true
			suite.Equal(expected, r)
			suite.NotEmpty(stack)
		}

		header1 = httpaux.NewHeaders("Custom1", "true")
		header2 = httpaux.NewHeaders("Custom2", "true")

		decorated = Middleware(
			WithStatusCode(588),
			WithOnRecover(onRecover1, onRecover2),
			WithRecoverBody(body),
			WithHeader(header1),
			WithHeader(header2),
		)(suite.panicHandler(expected))

		response = suite.serveDecoratedHTTP(decorated)
	)

	suite.Equal(588, response.Code)
	suite.Equal(customRecoverBody, response.Body.String())
	suite.Equal("true", response.Result().Header.Get("Custom1"))
	suite.Equal("true", response.Result().Header.Get("Custom2"))
	suite.True(onRecover1Called)
	suite.True(onRecover2Called)
	suite.Contains(response.Body.String(), customRecoverBody)
}

func (suite *MiddlewareSuite) TestEnrichedPanicValue() {
	var (
		expected = enrichedPanicValue{
			message:    "expected panic value",
			statusCode: 567,
			headers: http.Header{
				"Fubar": []string{"true"},
			},
		}

		decorated = Middleware()(
			suite.panicHandler(expected),
		)

		response = suite.serveDecoratedHTTP(decorated)
	)

	suite.Equal(567, response.Code)
	suite.Equal("true", response.Result().Header.Get("Fubar"))
	suite.Contains(response.Body.String(), expected.String())
}

func TestMiddleware(t *testing.T) {
	suite.Run(t, new(MiddlewareSuite))
}
