package client

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux"
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
	})(request)

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

// assertRequest verifies that the given client is functional
func (suite *ChainTestSuite) assertRequest(expectedOrder []int, client httpaux.Client) {
	suite.order = nil
	request, err := http.NewRequest("GET", suite.server.URL+"/test", nil)
	suite.Require().NoError(err)

	response, err := client.Do(request)
	suite.Require().NoError(err)

	defer response.Body.Close()
	io.Copy(ioutil.Discard, response.Body)
	suite.Equal(expectedOrder, suite.order, "the decorators did not run in the expected order")
	suite.Equal(299, response.StatusCode, "the test server was not invoked")
}

// newConstructor is a helper for returning a Constructor that expects its
// decorator to be in a certain order relative to other constructors
func (suite *ChainTestSuite) newConstructor(order int) Constructor {
	return func(next httpaux.Client) httpaux.Client {
		return Func(func(r *http.Request) (*http.Response, error) {
			suite.order = append(suite.order, order)
			return next.Do(r)
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

			suite.Run("WithClient", func() {
				client := new(http.Client)
				decorated := appended.Then(client)
				suite.Require().NotNil(decorated)
				suite.assertRequest(record.expectedOrder, decorated)
			})

			suite.Run("NilClient", func() {
				decorated := appended.Then(nil)
				suite.Require().NotNil(decorated)
				suite.assertRequest(record.expectedOrder, decorated)
			})

			suite.Run("WithClientFunc", func() {
				client := new(http.Client)
				decorated := appended.ThenFunc(client.Do)
				suite.Require().NotNil(decorated)
				suite.assertRequest(record.expectedOrder, decorated)
			})

			suite.Run("NilClientFunc", func() {
				decorated := appended.ThenFunc(nil)
				suite.Require().NotNil(decorated)
				suite.assertRequest(record.expectedOrder, decorated)
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

			suite.Run("WithClient", func() {
				client := new(http.Client)
				decorated := extended.Then(client)
				suite.Require().NotNil(decorated)
				suite.assertRequest(record.expectedOrder, decorated)
			})

			suite.Run("NilClient", func() {
				decorated := extended.Then(nil)
				suite.Require().NotNil(decorated)
				suite.assertRequest(record.expectedOrder, decorated)
			})

			suite.Run("WithClientFunc", func() {
				client := new(http.Client)
				decorated := extended.ThenFunc(client.Do)
				suite.Require().NotNil(decorated)
				suite.assertRequest(record.expectedOrder, decorated)
			})

			suite.Run("NilClientFunc", func() {
				decorated := extended.ThenFunc(nil)
				suite.Require().NotNil(decorated)
				suite.assertRequest(record.expectedOrder, decorated)
			})
		})
	}
}

func TestChain(t *testing.T) {
	suite.Run(t, new(ChainTestSuite))
}
