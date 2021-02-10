package httpmock

import (
	"bytes"
	"io"
	"io/ioutil"
)

// EmptyBody is a simpler way to invoke BodyBytes(nil)
func EmptyBody() io.ReadCloser {
	return BodyBytes(nil)
}

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
