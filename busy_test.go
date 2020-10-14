package httpbuddy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMaxRequestLimiterUnlimited(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		limiter MaxRequestLimiter
	)

	// just check that a nop function is returned and that Check always
	// allows requests.  *http.Request can be nil, as this limiter doesn't use the request

	first, ok := limiter.Check(nil)
	require.NotNil(first)
	assert.True(ok)

	second, ok := limiter.Check(nil)
	require.NotNil(second)
	assert.True(ok)

	second()
	third, ok := limiter.Check(nil)
	require.NotNil(third)
	assert.True(ok)

	first()
	third()
}

func testMaxRequestLimiterLimited(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		limiter = MaxRequestLimiter{
			MaxRequests: 1,
		}
	)

	first, ok := limiter.Check(nil)
	require.NotNil(first)
	assert.True(ok)

	second, ok := limiter.Check(nil)
	require.NotNil(second)
	assert.False(ok)

	// second should be a nop
	second()

	third, ok := limiter.Check(nil)
	require.NotNil(third)
	assert.False(ok)

	first()

	fourth, ok := limiter.Check(nil)
	assert.NotNil(fourth)
	assert.True(ok)
}

func TestMaxRequestLimiter(t *testing.T) {
	t.Run("Unlimited", testMaxRequestLimiterUnlimited)
	t.Run("Limited", testMaxRequestLimiterLimited)
}

func TestBusy(t *testing.T) {
}
