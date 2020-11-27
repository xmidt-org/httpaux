package roundtrip

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunc(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		expected = httptest.NewRequest("GET", "/", nil)

		called bool
		f      Func = func(actual *http.Request) (*http.Response, error) {
			called = true
			assert.Equal(expected, actual)
			return &http.Response{StatusCode: 211}, nil
		}
	)

	response, err := f.RoundTrip(expected)
	assert.True(called)
	assert.NoError(err)
	require.NotNil(response)
	assert.Equal(211, response.StatusCode)
}

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

	decorated := c.Then(expected)
	assert.True(called)
	require.NotNil(decorated)
	assert.Equal(expected, decorated)
}
