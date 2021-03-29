package httpaux

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/httpmock"
)

type HeaderTestSuite struct {
	suite.Suite
}

func TestHeader(t *testing.T) {
	suite.Run(t, new(HeaderTestSuite))
}

func testHeaderSetTo(t *testing.T, h Header, expected http.Header) {
	var (
		assert = assert.New(t)
		actual = http.Header{}
	)

	h.SetTo(actual)
	assert.Equal(expected, actual)
}

func testHeaderThen(t *testing.T, h Header, expected http.Header) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(271)
		})

		response  = httptest.NewRecorder()
		request   = httptest.NewRequest("GET", "/", nil)
		decorated = h.Then(handler)
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)

	assert.Equal(271, response.Code)
	assert.Equal(expected, response.Result().Header) // nolint:bodyclose
}

func testHeaderRoundTrip(t *testing.T, h Header, expected http.Header) {
	// TODO: this depends on a subpackage.  probably should refactor so that
	// doesn't happen
	var (
		assert  = assert.New(t)
		require = require.New(t)

		next      = httpmock.NewRoundTripper(t)
		request   = httptest.NewRequest("GET", "/", nil)
		decorated = h.RoundTrip(next)
	)

	next.OnRequest(request).Return(&http.Response{StatusCode: 284, Body: httpmock.EmptyBody()}, nil).Once()
	require.NotNil(decorated)
	response, err := decorated.RoundTrip(request)
	require.NotNil(response)
	defer response.Body.Close()
	assert.Equal(284, response.StatusCode)
	assert.NoError(err)

	next.AssertExpectations()
}

func testHeaderRoundTripDefault(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		decorated = NewHeaders("Test", "true").RoundTrip(nil)
	)

	require.NotNil(decorated)
	hrt, ok := decorated.(headerRoundTripper)
	require.True(ok)
	assert.Equal(http.DefaultTransport, hrt.next)
}

func _TestHeader(t *testing.T) {
	testData := []struct {
		header   Header
		expected http.Header
	}{
		{
			header:   NewHeader(nil),
			expected: http.Header{},
		},
		{
			header: NewHeader(http.Header{
				"Content-Type": {"text/plain"},
				"x-something":  {"value1", "value2"},
				"EMPTY":        {""},
			}),
			expected: http.Header{
				"Content-Type": {"text/plain"},
				"X-Something":  {"value1", "value2"},
				"Empty":        {""},
			},
		},
		{
			header:   NewHeaderFromMap(nil),
			expected: http.Header{},
		},
		{
			header: NewHeaderFromMap(map[string]string{
				"Content-Type": "text/plain",
				"x-something":  "value",
				"EmpTy":        "",
			}),
			expected: http.Header{
				"Content-Type": {"text/plain"},
				"X-Something":  {"value"},
				"Empty":        {""},
			},
		},
		{
			header:   NewHeaders(),
			expected: http.Header{},
		},
		{
			header: NewHeaders("Content-Type", "text/plain", "x-something", "value1", "EMptY", "", "x-sOMEthIng", "value2", "miSSing-valuE"),
			expected: http.Header{
				"Content-Type":  {"text/plain"},
				"X-Something":   {"value1", "value2"},
				"Empty":         {""},
				"Missing-Value": {""},
			},
		},
	}

	t.Run("SetTo", func(t *testing.T) {
		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testHeaderSetTo(t, record.header, record.expected)
			})
		}
	})

	t.Run("Then", func(t *testing.T) {
		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testHeaderThen(t, record.header, record.expected)
			})
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testHeaderRoundTrip(t, record.header, record.expected)
			})
		}
	})

	t.Run("RoundTripDefault", testHeaderRoundTripDefault)
}
