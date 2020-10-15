package httpaux

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGateAppend(t *testing.T) {
	testData := []struct {
		name string
		gate *Gate
	}{
		{
			name: "",
			gate: NewGate(),
		},
		{
			name: "test",
			gate: NewGateCustom("test", nil),
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert = assert.New(t)

				openedCalled bool
				opened       = func(n string) {
					openedCalled = true
					assert.Equal(record.name, n)
				}

				closedCalled bool
				closed       = func(n string) {
					closedCalled = true
					assert.Equal(record.name, n)
				}
			)

			assert.Equal(record.name, record.gate.Name())
			assert.True(record.gate.IsOpen())
			assert.NotEmpty(record.gate.String())

			record.gate.Append(GateHook{}) // nop
			assert.True(record.gate.IsOpen())

			record.gate.Append(GateHook{
				OnOpen:   opened,
				OnClosed: closed,
			})

			assert.True(record.gate.IsOpen())
			assert.True(openedCalled)
			assert.False(closedCalled)

			openedCalled = false
			closedCalled = false
			assert.True(record.gate.Close())
			assert.False(openedCalled)
			assert.True(closedCalled)

			openedCalled = false
			closedCalled = false
			assert.False(record.gate.Close())
			assert.False(openedCalled)
			assert.False(closedCalled)

			openedCalled = false
			closedCalled = false
			assert.True(record.gate.Open())
			assert.True(openedCalled)
			assert.False(closedCalled)

			openedCalled = false
			closedCalled = false
			assert.False(record.gate.Open())
			assert.False(openedCalled)
			assert.False(closedCalled)

			// test the case where the gate is closed when Append is called
			assert.True(record.gate.Close())
			openedCalled = false
			closedCalled = false
			record.gate.Append(GateHook{
				OnClosed: closed,
			})
			assert.False(openedCalled)
			assert.True(closedCalled)
		})
	}
}

func testGateThen(t *testing.T) {
	testData := []struct {
		name               string
		gate               *Gate
		expectedClosedCode int
	}{
		{
			name:               "",
			gate:               NewGate(),
			expectedClosedCode: http.StatusServiceUnavailable,
		},
		{
			name:               "test",
			gate:               NewGateCustom("test", ConstantHandler{StatusCode: http.StatusNotFound}),
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

				decorated = record.gate.Then(handler)
			)

			require.NotNil(decorated)

			response := httptest.NewRecorder()
			decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
			assert.Equal(217, response.Code)

			assert.True(record.gate.Close())
			response = httptest.NewRecorder()
			decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
			assert.Equal(record.expectedClosedCode, response.Code)

			assert.True(record.gate.Open())
			response = httptest.NewRecorder()
			decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
			assert.Equal(217, response.Code)
		})
	}
}

func testGateRoundTrip(t *testing.T) {
	testData := []struct {
		name               string
		gate               *Gate
		expectedClosedCode int
	}{
		{
			name:               "",
			gate:               NewGate(),
			expectedClosedCode: http.StatusServiceUnavailable,
		},
		{
			name:               "test",
			gate:               NewGateCustom("test", ConstantHandler{StatusCode: http.StatusNotFound}),
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

				decorated = record.gate.RoundTrip(roundTripper)
			)

			require.NotNil(decorated)

			response, err := decorated.RoundTrip(httptest.NewRequest("GET", "/", nil))
			assert.NoError(err)
			require.NotNil(response)
			assert.Equal(238, response.StatusCode)

			assert.True(record.gate.Close())
			response, err = decorated.RoundTrip(httptest.NewRequest("GET", "/", nil))
			require.Error(err)
			assert.Nil(response)
			assert.NotEmpty(err.Error())
			closedErr, ok := err.(*GateClosedError)
			require.True(ok)
			assert.Equal(record.gate, closedErr.Gate)

			assert.True(record.gate.Open())
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

		g = NewGate()
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
	t.Run("Append", testGateAppend)
	t.Run("Then", testGateThen)
	t.Run("RoundTrip", testGateRoundTrip)
	t.Run("DefaultTransport", testGateDefaultTransport)
}
