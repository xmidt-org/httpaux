package busy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/httpaux"
)

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

		decorated = Server{
			Limiter: limiter,
			Busy:    onBusy,
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

		handler   = httpaux.ConstantHandler{StatusCode: 222}
		decorated = Server{}.Then(handler)
	)

	require.NotNil(decorated)
	assert.Equal(handler, decorated)
}

func TestBusy(t *testing.T) {
	t.Run("Unlimited", testBusyUnlimited)

	t.Run("LimitedDefaultBusy", func(t *testing.T) {
		testBusyLimited(t, nil, http.StatusServiceUnavailable)
	})

	t.Run("LimitedCustomBusy", func(t *testing.T) {
		testBusyLimited(t, httpaux.ConstantHandler{StatusCode: 517}, 517)
	})
}
