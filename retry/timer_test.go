// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTimer(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		tc, stop = DefaultTimer(1 * time.Hour)
	)

	require.NotNil(tc)
	require.NotNil(stop)
	assert.True(stop())
	assert.False(stop())
}
