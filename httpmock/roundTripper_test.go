package httpmock

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRoundTripper(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		m       = new(RoundTripper)
		c       = http.Client{
			Transport: m,
		}

		expected = &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   "localhost",
				Path:   "/test",
			},
		}
	)

	m.OnRoundTrip(
		mock.MatchedBy(func(r *http.Request) bool {
			return r.URL.Path == "/test"
		}),
	).Return(&http.Response{StatusCode: 217}, nil).Once()

	r, err := c.Do(expected)
	assert.NoError(err)
	require.NotNil(r)
	assert.Equal(217, r.StatusCode)
	c.CloseIdleConnections()

	m.AssertExpectations(t)
}

func TestCloseIdler(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		m       = new(CloseIdler)
		c       = http.Client{
			Transport: m,
		}

		expected = &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   "localhost",
				Path:   "/testCloseIdler",
			},
		}
	)

	m.OnRoundTrip(
		mock.MatchedBy(func(r *http.Request) bool {
			return r.URL.Path == "/testCloseIdler"
		}),
	).Return(&http.Response{StatusCode: 288}, nil).Once()
	m.OnCloseIdleConnections().Once()

	r, err := c.Do(expected)
	assert.NoError(err)
	require.NotNil(r)
	assert.Equal(288, r.StatusCode)
	c.CloseIdleConnections()

	m.AssertExpectations(t)
}
