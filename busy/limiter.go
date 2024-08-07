// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package busy

import (
	"net/http"
	"sync/atomic"
)

// RequestDone is a callback that must be invoked when a request is
// finished so that its corresponding Limiter can update its state.
// Care must be taken by clients to invoke a RequestDone exactly once
// for any request.
type RequestDone func()

// NopRequestDone is a RequestDone implementation that does nothing.  Useful
// to use instead of nil.
func NopRequestDone() {}

// Limiter is a request limiter.  It constrains the number of concurrent
// requests either globally or based on some aspect of each request, such
// as a user account.
type Limiter interface {
	// Check enforces a concurrent request limit, possibly using information
	// from the given HTTP request.  If this method returns true, it indicates
	// that the request is allowed.  In that case, the RequestDone must be non-nil
	// and must be invoked by calling code to update the Limiter's state.
	//
	// If this method returns false, the request should not proceed.  The
	// RequestDone should be ignored.
	//
	// The RequestDone returned by this method is not guaranteed to be idempotent.
	// Callers must take care to invoke it exactly once for any given request.
	Check(*http.Request) (RequestDone, bool)
}

// MaxRequestLimiter is a Limiter that imposes a global limit for maximum
// concurrent requests.  No aspect of each HTTP request is taken into account.
type MaxRequestLimiter struct {
	// MaxRequests is the maximum number of concurrent HTTP requests.  If this
	// is nonpositive, then all requests are allowed.
	MaxRequests int64

	// counter is an atomically updated counter of current inflight requests
	counter int64
}

// release is the ReleaseDone for this instance.  it simply decrements the
// counter atomically.
func (rcl *MaxRequestLimiter) release() {
	atomic.AddInt64(&rcl.counter, -1)
}

// Check verifies that no more than MaxRequests requests are currently inflight.
// If MaxRequests is nonpositive, this method returns NopRequestDone and true.
func (rcl *MaxRequestLimiter) Check(*http.Request) (RequestDone, bool) {
	if rcl.MaxRequests < 1 {
		return NopRequestDone, true
	}

	count := atomic.AddInt64(&rcl.counter, 1)
	if count > rcl.MaxRequests {
		atomic.AddInt64(&rcl.counter, -1)
		return NopRequestDone, false
	}

	return rcl.release, true
}
