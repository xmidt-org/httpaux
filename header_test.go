package httpbuddy

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	assert.Equal(expected, response.HeaderMap)
}

func testHeaderRoundTrip(t *testing.T, h Header, expected http.Header) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		roundTripper = RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
			assert.Equal(expected, request.Header)
			return &http.Response{
				StatusCode: 284,
			}, nil
		})

		request   = httptest.NewRequest("GET", "/", nil)
		decorated = h.RoundTrip(roundTripper)
	)

	require.NotNil(decorated)
	response, err := decorated.RoundTrip(request)
	require.NotNil(response)
	assert.Equal(284, response.StatusCode)
	assert.NoError(err)
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

func TestHeader(t *testing.T) {
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
