package httpbuddy

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConstantHandlerDefault(t *testing.T) {
	var (
		assert = assert.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		ch ConstantHandler
	)

	ch.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Empty(response.HeaderMap)
	assert.Empty(response.Body.String())
}

func testConstantHandlerText(t *testing.T) {
	const text = "hello, world!"

	var (
		assert = assert.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		ch = ConstantHandler{
			StatusCode:  217,
			Header:      NewHeaders("Test", "true"),
			ContentType: "text/plain",
			Body:        []byte(text),
		}
	)

	ch.ServeHTTP(response, request)
	assert.Equal(217, response.Code)
	assert.Equal(
		http.Header{
			"Test":           {"true"},
			"Content-Type":   {"text/plain"},
			"Content-Length": {strconv.Itoa(len(text))},
		},
		response.HeaderMap,
	)

	assert.Equal(text, response.Body.String())
}

func testConstantHandlerNoContentType(t *testing.T) {
	const text = "hello, world!"

	var (
		assert = assert.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		ch = ConstantHandler{
			StatusCode: 217,
			Header:     NewHeaders("Test", "true"),
			Body:       []byte(text),
		}
	)

	ch.ServeHTTP(response, request)
	assert.Equal(217, response.Code)
	assert.Equal(
		http.Header{
			"Test":           {"true"},
			"Content-Length": {strconv.Itoa(len(text))},
		},
		response.HeaderMap,
	)

	assert.Equal(text, response.Body.String())
}

func TestConstantHandler(t *testing.T) {
	t.Run("Default", testConstantHandlerDefault)
	t.Run("Text", testConstantHandlerText)
	t.Run("NoContentType", testConstantHandlerNoContentType)
}

func TestConstantJSON(t *testing.T) {
	type Message struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		ch, err = ConstantJSON(Message{Name: "Joe Schmoe", Age: 152})
	)

	require.NoError(err)
	ch.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal(
		http.Header{
			"Content-Type":   {"application/json; charset=utf-8"},
			"Content-Length": {strconv.Itoa(len(ch.Body))},
		},
		response.HeaderMap,
	)

	assert.JSONEq(
		`{"name": "Joe Schmoe", "age": 152}`,
		response.Body.String(),
	)
}
