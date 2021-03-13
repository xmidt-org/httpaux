package retry

import (
	"context"
	"errors"
	"net/http"

	"github.com/xmidt-org/httpaux"
)

// Check is a predicate type used to determine if a result from http.Client.Do
// should be retried.  Implementations should only return true if it is reasonable
// to retry the request.  Examples of retryable situations are gateway timeouts
// and 429 (too many requests) responses.
//
// For any HTTP transaction that is considered a success, implementations should
// return false to halt further retries.
//
// For any HTTP transaction that failed but that shouldn't be retried, implementations
// should return false.  An example of this is an HTTP status of 400, since that indicates
// that no further use of that request will succeed.
type Check func(*http.Response, error) bool

// DefaultCheck is the Check predicate used if none is supplied.
//
// This default implementation returns true under the following conditions:
//
//   - The response is not nil and the status code is one of:
//       http.StatusRequestTimeout
//       http.StatusTooManyRequests
//       http.StatusGatewayTimeout
//   - The error is not nil and:
//       supplies a "Temporary() bool" method that returns true (including any wrapped errors)
//
// A consequence of honoring the Temporary() method on errors is that transient network errors
// will be retried.  Examples of this include DNS errors that are marked as temporary.
//
// In all other cases, this default function returns false.  Importantly, this means
// that context.Canceled errors are not retryable errors when using this check function.
func DefaultCheck(r *http.Response, err error) bool {
	if r != nil {
		switch r.StatusCode {
		case http.StatusRequestTimeout:
			return true

		case http.StatusTooManyRequests:
			return true

		case http.StatusGatewayTimeout:
			return true
		}
	}

	// allow for both the response and error to be non-nil by falling through.
	// unusual, but still technically allowed for a CheckRedirect.

	// context.DeadlineExceeded is a temporary error.
	// however, for this package we don't want to retry it by default because
	// it means the request context has expired.
	// see: https://go.googlesource.com/go/+/go1.16/src/context/context.go#167
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return httpaux.IsTemporary(err)
}
