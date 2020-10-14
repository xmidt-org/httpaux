package httpbuddy

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

// Busy defines a server middleware that enforces request limiting
type Busy struct {
	// Limiter is the concurrent request limiting strategy.  If this field is unset,
	// then no limiting is done.
	Limiter Limiter

	// OnBusy is the optional http.Handler to invoke when the maximum number
	// concurrent requests has been exceeded.  ConstantHandler is a useful choice
	// for this field, as it allows one to tailor not only the status code but also
	// the headers and body.
	//
	// If this field is nil, this middleware simply returns http.StatusServiceUnavailable.
	OnBusy http.Handler
}

// Then is a server middleware that enforces this Busy configuration.  If Limiter is nil,
// no decoration is done and next is returned as is.  If OnBusy is nil, then the returned
// handler will simply set http.StatusServiceUnavailable when requests fail the limit check.
func (b Busy) Then(next http.Handler) http.Handler {
	if b.Limiter == nil {
		return next
	}

	if b.OnBusy == nil {
		b.OnBusy = ConstantHandler{
			StatusCode: http.StatusServiceUnavailable,
		}
	}

	return &busyHandler{
		Busy: b,
		next: next,
	}
}

// busyHandler is a decorator around a next handler that enforces a request limit
type busyHandler struct {
	Busy
	next http.Handler
}

func (bh *busyHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if done, ok := bh.Limiter.Check(request); ok {
		defer done()
		bh.next.ServeHTTP(response, request)
	} else {
		bh.OnBusy.ServeHTTP(response, request)
	}
}
