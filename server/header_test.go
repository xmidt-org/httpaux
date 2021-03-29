package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/httpaux"
)

func TestHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		h = httpaux.NewHeaders("Header1", "value1")

		handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(299)
		})

		decorated = Header(h.SetTo)(handler)

		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)

	result := response.Result() //nolint:bodyclose
	assert.Equal(299, result.StatusCode)
	assert.Equal("value1", response.Header().Get("Header1"))
}
