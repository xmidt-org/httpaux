package roundtrip

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	decorated := c.Then(expected)
	assert.True(called)
	require.NotNil(decorated)
	assert.Equal(expected, decorated)
}
