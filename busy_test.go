package httpaux

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

		waiting  = make(chan struct{})
		unpaused = make(chan struct{})

		handlerPause = make(chan struct{}, 1)
		handler      = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			<-handlerPause
			response.WriteHeader(298)
		})

		decorated = Busy{
			Limiter: &MaxRequestLimiter{
				MaxRequests: 1,
			},
			OnBusy: onBusy,
		}.Then(handler)
	)

	require.NotNil(decorated)

	handlerPause <- struct{}{}
	first := httptest.NewRecorder()
	decorated.ServeHTTP(first, httptest.NewRequest("GET", "/", nil))
	assert.Equal(298, first.Code)

	go func() {
		defer close(unpaused)

		// pause this request to simulate a long-running ServeHTTP call
		response := httptest.NewRecorder()
		close(waiting)
		decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
		assert.Equal(298, response.Code)
	}()

	select {
	case <-waiting:
		// passing
	case <-time.After(time.Second):
		require.Fail("goroutine did not start waiting")
	}

	second := httptest.NewRecorder()
	decorated.ServeHTTP(second, httptest.NewRequest("GET", "/", nil))
	assert.Equal(expectedBusyCode, second.Code)

	handlerPause <- struct{}{}
	select {
	case <-unpaused:
		// passing
	case <-time.After(time.Second):
		require.Fail("the paused request did not complete")
	}

	handlerPause <- struct{}{}
	third := httptest.NewRecorder()
	decorated.ServeHTTP(third, httptest.NewRequest("GET", "/", nil))
	assert.Equal(298, third.Code)
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
