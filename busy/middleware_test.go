// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package busy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux"
)

type ServerTestSuite struct {
	suite.Suite
	decorated http.Handler
	limiter   *MaxRequestLimiter
}

var _ suite.SetupTestSuite = (*ServerTestSuite)(nil)

func (suite *ServerTestSuite) SetupTest() {
	suite.decorated = httpaux.ConstantHandler{
		StatusCode: 222,
	}

	suite.limiter = &MaxRequestLimiter{
		MaxRequests: 1,
	}
}

func (suite *ServerTestSuite) TestLimited() {
	decorator := Server{
		Limiter: suite.limiter,
	}.Then(suite.decorated)

	suite.Require().NotNil(decorator)

	first := httptest.NewRecorder()
	decorator.ServeHTTP(first, httptest.NewRequest("GET", "/", nil))
	suite.Equal(222, first.Code)
	suite.Zero(suite.limiter.counter)

	// simulate an overloaded handler
	suite.limiter.counter = 1
	second := httptest.NewRecorder()
	decorator.ServeHTTP(second, httptest.NewRequest("GET", "/", nil))
	suite.Equal(http.StatusServiceUnavailable, second.Code)
	suite.Equal(int64(1), suite.limiter.counter)
}

func (suite *ServerTestSuite) TestCustomBusy() {
	decorator := Server{
		Limiter: suite.limiter,
		Busy: httpaux.ConstantHandler{
			StatusCode: 599,
		},
	}.Then(suite.decorated)

	suite.Require().NotNil(decorator)

	first := httptest.NewRecorder()
	decorator.ServeHTTP(first, httptest.NewRequest("GET", "/", nil))
	suite.Equal(222, first.Code)
	suite.Zero(suite.limiter.counter)

	// simulate an overloaded handler
	suite.limiter.counter = 1
	second := httptest.NewRecorder()
	decorator.ServeHTTP(second, httptest.NewRequest("GET", "/", nil))
	suite.Equal(599, second.Code)
	suite.Equal(int64(1), suite.limiter.counter)
}

func (suite *ServerTestSuite) TestUnlimited() {
	decorator := Server{}.Then(suite.decorated)
	suite.Require().NotNil(decorator)
	suite.Equal(suite.decorated, decorator)
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
