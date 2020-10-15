package httpaux

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGateCallbacksInitiallyOpen(t *testing.T) {
	for _, count := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				options = GateOptions{
					Name: "test",
				}

				actualOnOpen   []int
				expectedOnOpen []int

				actualOnClosed   []int
				expectedOnClosed []int
			)

			for i := 0; i < count; i++ {
				i := i

				expectedOnOpen = append(expectedOnOpen, i)
				expectedOnClosed = append(expectedOnClosed, i)

				options.OnOpen = append(options.OnOpen, func(actual *Gate) {
					actualOnOpen = append(actualOnOpen, i)
					assert.Equal("test", actual.Name())
				})

				options.OnClosed = append(options.OnClosed, func(actual *Gate) {
					actualOnClosed = append(actualOnClosed, i)
					assert.Equal("test", actual.Name())
				})
			}

			g := NewGate(options)
			require.NotNil(g)
			assert.True(g.IsOpen())
			assert.Contains(g.String(), gateOpenText)

			// initial callbacks should be called
			assert.Equal(expectedOnOpen, actualOnOpen)
			assert.Empty(actualOnClosed)

			// Open should be idempotent
			actualOnOpen = nil
			actualOnClosed = nil
			assert.False(g.Open())
			assert.True(g.IsOpen())
			assert.Contains(g.String(), gateOpenText)
			assert.Empty(actualOnOpen)
			assert.Empty(actualOnClosed)

			// close callbacks should be called
			actualOnOpen = nil
			actualOnClosed = nil
			assert.True(g.Close())
			assert.False(g.IsOpen())
			assert.Contains(g.String(), gateClosedText)
			assert.Empty(actualOnOpen)
			assert.Equal(expectedOnClosed, actualOnClosed)

			// Close should be idempotent
			actualOnOpen = nil
			actualOnClosed = nil
			assert.False(g.Close())
			assert.False(g.IsOpen())
			assert.Contains(g.String(), gateClosedText)
			assert.Empty(actualOnOpen)
			assert.Empty(actualOnClosed)
		})
	}
}

func testGateCallbacksInitiallyClosed(t *testing.T) {
	for _, count := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				options = GateOptions{
					Name:            "test",
					InitiallyClosed: true,
				}

				actualOnOpen   []int
				expectedOnOpen []int

				actualOnClosed   []int
				expectedOnClosed []int
			)

			for i := 0; i < count; i++ {
				i := i

				expectedOnOpen = append(expectedOnOpen, i)
				expectedOnClosed = append(expectedOnClosed, i)

				options.OnOpen = append(options.OnOpen, func(actual *Gate) {
					actualOnOpen = append(actualOnOpen, i)
					assert.Equal("test", actual.Name())
				})

				options.OnClosed = append(options.OnClosed, func(actual *Gate) {
					actualOnClosed = append(actualOnClosed, i)
					assert.Equal("test", actual.Name())
				})
			}

			g := NewGate(options)
			require.NotNil(g)
			assert.False(g.IsOpen())
			assert.Contains(g.String(), gateClosedText)

			// initial callbacks should be called
			assert.Empty(actualOnOpen)
			assert.Equal(expectedOnClosed, actualOnClosed)

			// Close should be idempotent
			actualOnOpen = nil
			actualOnClosed = nil
			assert.False(g.Close())
			assert.False(g.IsOpen())
			assert.Contains(g.String(), gateClosedText)
			assert.Empty(actualOnOpen)
			assert.Empty(actualOnClosed)

			// open callbacks should be called
			actualOnOpen = nil
			actualOnClosed = nil
			assert.True(g.Open())
			assert.True(g.IsOpen())
			assert.Contains(g.String(), gateOpenText)
			assert.Equal(expectedOnOpen, actualOnOpen)
			assert.Empty(actualOnClosed)

			// Open should be idempotent
			actualOnOpen = nil
			actualOnClosed = nil
			assert.False(g.Open())
			assert.True(g.IsOpen())
			assert.Contains(g.String(), gateOpenText)
			assert.Empty(actualOnOpen)
			assert.Empty(actualOnClosed)
		})
	}
}

func testGateThen(t *testing.T) {
	testData := []struct {
		options            GateOptions
		expectedClosedCode int
	}{
		{
			options:            GateOptions{},
			expectedClosedCode: http.StatusServiceUnavailable,
		},
		{
			options: GateOptions{
				Name:          "test",
				ClosedHandler: ConstantHandler{StatusCode: http.StatusNotFound},
			},
			expectedClosedCode: http.StatusNotFound,
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					response.WriteHeader(217)
				})

				g = NewGate(record.options)
			)

			require.NotNil(g)
			decorated := g.Then(handler)
			require.NotNil(decorated)

			response := httptest.NewRecorder()
			decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
			assert.Equal(217, response.Code)

			assert.True(g.Close())
			response = httptest.NewRecorder()
			decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
			assert.Equal(record.expectedClosedCode, response.Code)

			assert.True(g.Open())
			response = httptest.NewRecorder()
			decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
			assert.Equal(217, response.Code)
		})
	}
}

func testGateRoundTrip(t *testing.T) {
	testData := []struct {
		options            GateOptions
		expectedClosedCode int
	}{
		{
			options:            GateOptions{},
			expectedClosedCode: http.StatusServiceUnavailable,
		},
		{
			options: GateOptions{
				Name:          "test",
				ClosedHandler: ConstantHandler{StatusCode: http.StatusNotFound},
			},
			expectedClosedCode: http.StatusNotFound,
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				roundTripper = RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 238,
					}, nil
				})

				g = NewGate(record.options)
			)

			require.NotNil(g)
			decorated := g.RoundTrip(roundTripper)
			require.NotNil(decorated)

			response, err := decorated.RoundTrip(httptest.NewRequest("GET", "/", nil))
			assert.NoError(err)
			require.NotNil(response)
			assert.Equal(238, response.StatusCode)

			assert.True(g.Close())
			response, err = decorated.RoundTrip(httptest.NewRequest("GET", "/", nil))
			require.Error(err)
			assert.Nil(response)
			assert.NotEmpty(err.Error())
			closedErr, ok := err.(*GateClosedError)
			require.True(ok)
			assert.Equal(g, closedErr.Gate)

			assert.True(g.Open())
			response, err = decorated.RoundTrip(httptest.NewRequest("GET", "/", nil))
			assert.NoError(err)
			require.NotNil(response)
			assert.Equal(238, response.StatusCode)
		})
	}
}

func testGateDefaultTransport(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		g = NewGate(GateOptions{})
	)

	require.NotNil(g)

	roundTripper := g.RoundTrip(nil)
	require.NotNil(roundTripper)

	assert.True(g.Close())
	response, err := roundTripper.RoundTrip(httptest.NewRequest("GET", "/", nil))
	assert.Nil(response)
	require.NotNil(err)
	assert.NotEmpty(err.Error())

	closedErr, ok := err.(*GateClosedError)
	require.True(ok)
	assert.Equal(g, closedErr.Gate)
}

func TestGate(t *testing.T) {
	t.Run("Callbacks", func(t *testing.T) {
		t.Run("InitiallyOpen", testGateCallbacksInitiallyOpen)
		t.Run("InitiallyClosed", testGateCallbacksInitiallyClosed)
	})

	t.Run("Then", testGateThen)
	t.Run("RoundTrip", testGateRoundTrip)
	t.Run("DefaultTransport", testGateDefaultTransport)
}
