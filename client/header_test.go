// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package client

import (
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
		c  = &http.Client{
			Transport: rt,
		}

		decorated = Header(h.SetTo)(c)
		expected  = new(http.Response)
	)

	request, err := http.NewRequest("GET", "/", nil)
	require.NoError(err)

	require.NotNil(decorated)
	rt.OnAny().AssertRequest(
		httpmock.Header("Header1", "value1"),
	).Return(expected, nil).Once()

	actual, actualErr := decorated.Do(request) //nolint:bodyclose
	assert.True(expected == actual)
	assert.NoError(actualErr)

	rt.AssertExpectations()
}
