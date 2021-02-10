package httpmock

import (
	"bytes"
	"io"
	"io/ioutil"
)

// BodyString is syntactic sugar for creating a response body from a string
func BodyString(b string) io.ReadCloser {
	return ioutil.NopCloser(
		bytes.NewBufferString(b),
	)
}

// BodyBytes is syntactic sugar for creating a response body from a byte slice
func BodyBytes(b []byte) io.ReadCloser {
	return ioutil.NopCloser(
		bytes.NewBuffer(b),
	)
}
