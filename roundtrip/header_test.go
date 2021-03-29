package roundtrip

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/httpmock"
)

func TestHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		h  = httpaux.NewHeaders("Header1", "value1")
		rt = httpmock.NewRoundTripper(t)

		decorated = Header(h.SetTo)(rt)

		request = &http.Request{
			Header: make(http.Header),
		}

		expected    = new(http.Response)
		expectedErr = errors.New("expected")
	)

	require.NotNil(decorated)
	rt.OnAny().AssertRequest(
		httpmock.Header("Header1", "value1"),
	).Return(expected, expectedErr).Once()

	actual, actualErr := decorated.RoundTrip(request)
	assert.True(expected == actual)
	assert.True(expectedErr == actualErr)

	rt.AssertExpectations()
}
