// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package erraux

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testErrorSimple(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		wrappedErr = errors.New("expected")
		err        = &Error{
			Err: wrappedErr,
		}
	)

	assert.Equal(wrappedErr, err.Unwrap())
	assert.Equal(http.StatusInternalServerError, err.StatusCode())
	assert.Empty(err.Headers())
	assert.Contains(err.Error(), "expected")

	msg, marshalErr := json.Marshal(err)
	require.NoError(marshalErr)
	assert.JSONEq(
		`{"code": 500, "cause": "expected"}`,
		string(msg),
	)
}

func testErrorNoMessage(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		wrappedErr = errors.New("expected")
		err        = &Error{
			Err:  wrappedErr,
			Code: http.StatusNotFound,
			Header: http.Header{
				"Error": {"value"},
			},
		}
	)

	assert.Equal(wrappedErr, err.Unwrap())
	assert.Equal(http.StatusNotFound, err.StatusCode())
	assert.Equal(
		http.Header{
			"Error": {"value"},
		},
		err.Headers(),
	)

	assert.Contains(err.Error(), "expected")

	msg, marshalErr := json.Marshal(err)
	require.NoError(marshalErr)
	assert.JSONEq(
		`{"code": 404, "cause": "expected"}`,
		string(msg),
	)
}

func testErrorCustomMessage(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		wrappedErr = errors.New("expected")
		err        = &Error{
			Err:     wrappedErr,
			Message: "test",
			Code:    http.StatusNotFound,
			Header: http.Header{
				"Error": {"value"},
			},
		}
	)

	assert.Equal(wrappedErr, err.Unwrap())
	assert.Equal(http.StatusNotFound, err.StatusCode())
	assert.Equal(
		http.Header{
			"Error": {"value"},
		},
		err.Headers(),
	)

	assert.Contains(err.Error(), "expected")
	assert.Contains(err.Error(), "test")

	msg, marshalErr := json.Marshal(err)
	require.NoError(marshalErr)
	assert.JSONEq(
		`{"code": 404, "message": "test", "cause": "expected"}`,
		string(msg),
	)
}

func TestError(t *testing.T) {
	t.Run("Simple", testErrorSimple)
	t.Run("NoMessage", testErrorNoMessage)
	t.Run("CustomMessage", testErrorCustomMessage)
}

func TestIsTemporary(t *testing.T) {
	assert := assert.New(t)

	assert.False(
		IsTemporary(errors.New("this isn't a temporary error")),
	)

	assert.False(
		IsTemporary(&net.DNSError{
			IsTemporary: false,
		}),
	)

	assert.True(
		IsTemporary(&net.DNSError{
			IsTemporary: true,
		}),
	)

	assert.False(
		IsTemporary(context.Canceled),
	)

	// context.DeadlineExceeded is a Temporary error
	// see: https://go.googlesource.com/go/+/go1.16/src/context/context.go#167
	assert.True(
		IsTemporary(context.DeadlineExceeded),
	)
}
