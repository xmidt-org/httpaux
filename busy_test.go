package httpaux

import (
	"net/http"
	"net/http/httptest"
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

func testBusyLimited(t *testing.T, onBusy http.Handler, expectedBusyCode int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(298)
		})

		limiter = &MaxRequestLimiter{
			MaxRequests: 1,
		}

		decorated = Busy{
			Limiter: limiter,
			OnBusy:  onBusy,
		}.Then(handler)
	)

	require.NotNil(decorated)

	first := httptest.NewRecorder()
	decorated.ServeHTTP(first, httptest.NewRequest("GET", "/", nil))
	assert.Equal(298, first.Code)
	assert.Zero(limiter.counter)

	// simulate an overloaded handler
	limiter.counter = 1
	second := httptest.NewRecorder()
	decorated.ServeHTTP(second, httptest.NewRequest("GET", "/", nil))
	assert.Equal(expectedBusyCode, second.Code)
	assert.Equal(int64(1), limiter.counter)
}

func testBusyUnlimited(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		handler   = ConstantHandler{StatusCode: 222}
		decorated = Busy{}.Then(handler)
	)

	require.NotNil(decorated)
	assert.Equal(handler, decorated)
}

func TestBusy(t *testing.T) {
	t.Run("Unlimited", testBusyUnlimited)

	t.Run("LimitedDefaultOnBusy", func(t *testing.T) {
		testBusyLimited(t, nil, http.StatusServiceUnavailable)
	})

	t.Run("LimitedCustomOnBusy", func(t *testing.T) {
		testBusyLimited(t, ConstantHandler{StatusCode: 517}, 517)
	})
}
