package httpmock

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyBody(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		body = EmptyBody()
	)

	require.NotNil(body)
	b, err := ioutil.ReadAll(body)
	assert.Empty(b)
	assert.NoError(err)
	assert.NoError(body.Close())
}

func TestBodyBytes(t *testing.T) {
	const bodyContents = "some lovely content here"

	var (
		assert  = assert.New(t)
		require = require.New(t)

		body = BodyBytes([]byte(bodyContents))
	)

	require.NotNil(body)
	b, err := ioutil.ReadAll(body)
	assert.Equal(bodyContents, string(b))
	assert.NoError(err)
	assert.NoError(body.Close())
}

func TestBodyString(t *testing.T) {
	const bodyContents = "some lovely content here"

	var (
		assert  = assert.New(t)
		require = require.New(t)

		body = BodyString(bodyContents)
	)

	require.NotNil(body)
	b, err := ioutil.ReadAll(body)
	assert.Equal(bodyContents, string(b))
	assert.NoError(err)
	assert.NoError(body.Close())
}
