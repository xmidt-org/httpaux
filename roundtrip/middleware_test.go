package roundtrip

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/httpmock"
)

func TestConstructor(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		expected = &http.Transport{
			MaxResponseHeaderBytes: 1234,
		}

		called bool
		c      Constructor = func(actual http.RoundTripper) http.RoundTripper {
			called = true
			assert.Equal(expected, actual)
			return actual
		}
	)

	decorated := c.ThenRoundTrip(expected)
	assert.True(called)
	require.NotNil(decorated)
	assert.Equal(expected, decorated)
}

type ChainTestSuite struct {
	suite.Suite
}

func (suite *ChainTestSuite) TestUninitialized() {
	var (
		next  = new(httpmock.RoundTripper)
		chain Chain

		decorator = chain.ThenRoundTrip(next)
	)

	suite.Require().NotNil(decorator)
	suite.Equal(next, decorator)
	next.AssertExpectations(suite.T())
}

func (suite *ChainTestSuite) TestEmpty() {
	var (
		next  = new(httpmock.RoundTripper)
		chain = NewChain()

		decorator = chain.ThenRoundTrip(next)
	)

	suite.Require().NotNil(decorator)
	suite.Equal(next, decorator)
	next.AssertExpectations(suite.T())
}

func (suite *ChainTestSuite) TestAppend() {
	var (
		next    = new(httpmock.RoundTripper)
		called  []int
		initial = NewChain(
			func(next http.RoundTripper) http.RoundTripper {
				called = append(called, 0)
				return next
			},
			func(next http.RoundTripper) http.RoundTripper {
				called = append(called, 1)
				return next
			},
		)
	)

	suite.Equal(initial, initial.Append())
	suite.Same(next, initial.ThenRoundTrip(next))
	suite.Equal([]int{1, 0}, called)

	appended := initial.Append(
		func(next http.RoundTripper) http.RoundTripper {
			called = append(called, 2)
			return next
		},
	)

	called = nil
	suite.NotEqual(initial, appended)
	suite.Same(next, appended.ThenRoundTrip(next))
	suite.Equal([]int{2, 1, 0}, called)

	// initial shouldn't have changed
	called = nil
	suite.Same(next, initial.ThenRoundTrip(next))
	suite.Equal([]int{1, 0}, called)

	next.AssertExpectations(suite.T())
}

func (suite *ChainTestSuite) TestExtend() {
	var (
		next    = new(httpmock.RoundTripper)
		called  []int
		initial = NewChain(
			func(next http.RoundTripper) http.RoundTripper {
				called = append(called, 0)
				return next
			},
			func(next http.RoundTripper) http.RoundTripper {
				called = append(called, 1)
				return next
			},
		)
	)

	suite.Equal(initial, initial.Extend(Chain{}))
	suite.Same(next, initial.ThenRoundTrip(next))
	suite.Equal([]int{1, 0}, called)

	extended := initial.Extend(
		NewChain(
			func(next http.RoundTripper) http.RoundTripper {
				called = append(called, 2)
				return next
			},
		),
	)

	called = nil
	suite.NotEqual(initial, extended)
	suite.Same(next, extended.ThenRoundTrip(next))
	suite.Equal([]int{2, 1, 0}, called)

	// initial shouldn't have changed
	called = nil
	suite.Same(next, initial.ThenRoundTrip(next))
	suite.Equal([]int{1, 0}, called)

	next.AssertExpectations(suite.T())
}

func (suite *ChainTestSuite) TestThenDefaultTransport() {
	chain := NewChain(
		func(next http.RoundTripper) http.RoundTripper {
			return next
		},
	)

	suite.Same(http.DefaultTransport, chain.ThenRoundTrip(nil))
}

func (suite *ChainTestSuite) TestThenNoCloseIdler() {
	var (
		request  = httptest.NewRequest("GET", "/noCloseIdler", nil)
		response = &http.Response{
			StatusCode: 674,
			Body:       httpmock.EmptyBody(),
		}
		err = errors.New("expected no CloseIdler error")

		next   = new(httpmock.RoundTripper)
		called []int
		chain  = NewChain(
			func(next http.RoundTripper) http.RoundTripper {
				return Func(func(r *http.Request) (*http.Response, error) {
					called = append(called, 0)
					return next.RoundTrip(r)
				})
			},
			func(next http.RoundTripper) http.RoundTripper {
				return Func(func(r *http.Request) (*http.Response, error) {
					called = append(called, 1)
					return next.RoundTrip(r)
				})
			},
		)
	)

	decorator := chain.ThenRoundTrip(next)
	suite.Require().NotNil(decorator)
	next.OnRoundTrip(request).Once().Return(response, err)

	actual, actualErr := decorator.RoundTrip(request)
	suite.Require().NotNil(actual)
	defer actual.Body.Close()
	suite.Equal(response, actual)
	suite.Equal(err, actualErr)
	suite.Equal([]int{0, 1}, called)

	_, ok := decorator.(CloseIdler)
	suite.False(ok)
	next.AssertExpectations(suite.T())
}

func (suite *ChainTestSuite) TestThenCloseIdler() {
	var (
		request  = httptest.NewRequest("GET", "/closeIdler", nil)
		response = &http.Response{
			StatusCode: 722,
			Body:       httpmock.EmptyBody(),
		}
		err = errors.New("expected CloseIdler error")

		next   = new(httpmock.CloseIdler)
		called []int
		chain  = NewChain(
			func(next http.RoundTripper) http.RoundTripper {
				return Func(func(r *http.Request) (*http.Response, error) {
					called = append(called, 0)
					return next.RoundTrip(r)
				})
			},
			func(next http.RoundTripper) http.RoundTripper {
				return Func(func(r *http.Request) (*http.Response, error) {
					called = append(called, 1)
					return next.RoundTrip(r)
				})
			},
		)
	)

	decorator := chain.ThenRoundTrip(next)
	suite.Require().NotNil(decorator)
	next.OnRoundTrip(request).Once().Return(response, err)
	next.OnCloseIdleConnections().Once()

	actual, actualErr := decorator.RoundTrip(request)
	suite.Require().NotNil(actual)
	defer actual.Body.Close()
	suite.Equal(response, actual)
	suite.Equal(err, actualErr)
	suite.Equal([]int{0, 1}, called)

	suite.Require().Implements((*CloseIdler)(nil), decorator)
	decorator.(CloseIdler).CloseIdleConnections()

	next.AssertExpectations(suite.T())
}

func TestChain(t *testing.T) {
	suite.Run(t, new(ChainTestSuite))
}
