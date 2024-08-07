// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package roundtrip

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestFunc(t *testing.T) {
	var (
		assert = assert.New(t)

		request     = new(http.Request)
		expected    = new(http.Response)
		expectedErr = errors.New("expected")
	)

	//nolint:bodyclose
	actual, actualErr := Func(func(r *http.Request) (*http.Response, error) {
		assert.True(request == r)
		return expected, expectedErr
	}).RoundTrip(request)

	assert.True(expected == actual)
	assert.Equal(expectedErr, actualErr)
}

type ChainTestSuite struct {
	suite.Suite

	server *httptest.Server

	// order is used to verify the execution order of decorators
	order []int
}

var _ suite.SetupTestSuite = (*ChainTestSuite)(nil)
var _ suite.SetupAllSuite = (*ChainTestSuite)(nil)
var _ suite.TearDownAllSuite = (*ChainTestSuite)(nil)

func (suite *ChainTestSuite) testHandle(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(299)
}

func (suite *ChainTestSuite) SetupSuite() {
	suite.server = httptest.NewServer(
		http.HandlerFunc(suite.testHandle),
	)
}

func (suite *ChainTestSuite) SetupTest() {
	suite.order = nil
}

func (suite *ChainTestSuite) TearDownSuite() {
	suite.server.Close()
	suite.server = nil
}

// assertRequest verifies that the given round tripper is functional
func (suite *ChainTestSuite) assertRequest(expectedOrder []int, transport http.RoundTripper) {
	suite.Require().NotNil(transport)

	suite.order = nil
	request, err := http.NewRequest("GET", suite.server.URL+"/test", nil)
	suite.Require().NoError(err)

	client := &http.Client{
		Transport: transport,
	}

	response, err := client.Do(request)
	suite.Require().NoError(err)

	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)
	suite.Equal(expectedOrder, suite.order, "the decorators did not run in the expected order")
	suite.Equal(299, response.StatusCode, "the test server was not invoked")
}

// newConstructor is a helper for returning a Constructor that expects its
// decorator to be in a certain order relative to other constructors
func (suite *ChainTestSuite) newConstructor(order int) Constructor {
	return func(next http.RoundTripper) http.RoundTripper {
		return Func(func(r *http.Request) (*http.Response, error) {
			suite.order = append(suite.order, order)
			return next.RoundTrip(r)
		})
	}
}

func (suite *ChainTestSuite) TestAppend() {
	testData := []struct {
		chain         Chain
		toAppend      []Constructor
		expectedOrder []int
	}{
		{
			chain:         Chain{},
			toAppend:      nil,
			expectedOrder: nil,
		},
		{
			chain: NewChain(
				suite.newConstructor(0),
			),
			expectedOrder: []int{0},
		},
		{
			chain: Chain{},
			toAppend: []Constructor{
				suite.newConstructor(0),
			},
			expectedOrder: []int{0},
		},
		{
			chain: NewChain(
				suite.newConstructor(0),
				suite.newConstructor(1),
			),
			toAppend: []Constructor{
				suite.newConstructor(2),
				suite.newConstructor(3),
			},
			expectedOrder: []int{0, 1, 2, 3},
		},
	}

	for i, record := range testData {
		suite.Run(strconv.Itoa(i), func() {
			appended := record.chain.Append(record.toAppend...)

			suite.Run("WithRoundTripper", func() {
				suite.assertRequest(
					record.expectedOrder,
					appended.Then(new(http.Transport)),
				)
			})

			suite.Run("NilRoundTripper", func() {
				suite.assertRequest(
					record.expectedOrder,
					appended.Then(nil),
				)
			})

			suite.Run("WithRoundTripperFunc", func() {
				transport := new(http.Transport)
				suite.assertRequest(
					record.expectedOrder,
					appended.ThenFunc(transport.RoundTrip),
				)
			})

			suite.Run("NilRoundTripperFunc", func() {
				suite.assertRequest(
					record.expectedOrder,
					appended.ThenFunc(nil),
				)
			})
		})
	}
}

func (suite *ChainTestSuite) TestExtend() {
	testData := []struct {
		chain         Chain
		toExtend      Chain
		expectedOrder []int
	}{
		{
			chain:         Chain{},
			toExtend:      Chain{},
			expectedOrder: nil,
		},
		{
			chain: NewChain(
				suite.newConstructor(0),
			),
			expectedOrder: []int{0},
		},
		{
			chain: Chain{},
			toExtend: NewChain(
				suite.newConstructor(0),
			),
			expectedOrder: []int{0},
		},
		{
			chain: NewChain(
				suite.newConstructor(0),
				suite.newConstructor(1),
			),
			toExtend: NewChain(
				suite.newConstructor(2),
				suite.newConstructor(3),
			),
			expectedOrder: []int{0, 1, 2, 3},
		},
	}

	for i, record := range testData {
		suite.Run(strconv.Itoa(i), func() {
			extended := record.chain.Extend(record.toExtend)

			suite.Run("WithRoundTripper", func() {
				suite.assertRequest(
					record.expectedOrder,
					extended.Then(new(http.Transport)),
				)
			})

			suite.Run("NilRoundTripper", func() {
				suite.assertRequest(
					record.expectedOrder,
					extended.Then(nil),
				)
			})

			suite.Run("WithRoundTripperFunc", func() {
				transport := new(http.Transport)
				suite.assertRequest(
					record.expectedOrder,
					extended.ThenFunc(transport.RoundTrip),
				)
			})

			suite.Run("NilRoundTripperFunc", func() {
				suite.assertRequest(
					record.expectedOrder,
					extended.ThenFunc(nil),
				)
			})
		})
	}
}

func TestChain(t *testing.T) {
	suite.Run(t, new(ChainTestSuite))
}
