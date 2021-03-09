package retry

import (
	"context"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/client"
)

// NoGetBodyError indicates that the initial attempt at an HTTP transaction required
// a retry but that the *http.Request had no GetBody field.
//
// If an initial attempt succeeded (i.e. did NOT require a retry), the GetBody field
// is ignored.  This error only occurs if retries were required.
type NoGetBodyError struct {
	// Initial is the error from the initial Do(request)
	Initial error
}

// Error fulfills the error interface
func (err *NoGetBodyError) Error() string {
	return "http.Request.GetBody must be set in order to retry the transaction"
}

// GetBodyError indicates that http.Request.GetBody returned an error.  Retries
// cannot continue in this case, since the original request body is unavailable.
type GetBodyError struct {
	// Err is the error returned from GetBody
	Err error
}

// Error fulfills the error interface
func (err *GetBodyError) Error() string {
	var o strings.Builder
	o.WriteString("GetBody returned an error: [")
	o.WriteString(err.Err.Error())
	o.WriteRune(']')
	return o.String()
}

// New creates a middleware constructor that decorates an HTTP client
// for retries.  If cfg.Retries is nonpositive, the returned constructor does
// no decoration.  This function is the primary and recommended way
// to implement HTTP client retry behavior.
//
// The returned constructor will decorate http.DefaultClient if passed a nil
// client.
func New(cfg Config) client.Constructor {
	prototype := NewClient(cfg, nil)
	if prototype == nil {
		return func(next httpaux.Client) httpaux.Client {
			return next
		}
	}

	return func(next httpaux.Client) httpaux.Client {
		// now we can just clone the prototype and set the next member
		c := new(Client)
		*c = *prototype

		// if next == nil, then we leave the next set to http.DefaultClient
		if next != nil {
			c.next = next
		}

		return c
	}
}

// Client is an httpaux.Client that retries HTTP transactions.  The middleware
// returned by New uses this type to perform retries.
type Client struct {
	// next is the decorated client used to execute HTTP transactions
	next httpaux.Client

	// maxElapsedTime is used for the parent context for all retries
	// if this value is nonpositive, no enforcement of max elapsed time is done
	maxElapsedTime time.Duration

	// intervals is the precomputed retry intervals, including jitter
	intervals intervals

	// random is the source of randomness
	random Random

	// timer is the Timer strategy used to wait between retries
	timer Timer

	// check is the Check strategy for determining if a request should be retried
	check Check
}

// NewClient constructs a Client from a configuration.  If cfg.Retries
// is nonpositive, this function returns nil.
//
// The next instance is used to actually execute HTTP transactions.  If next
// is nil, http.DefaultClient is used.
//
// This function is a low-level way to instantiate a client.  Most users
// will instead create a middleware with New.  This function can be used
// for more detailed usage.
func NewClient(cfg Config, next httpaux.Client) *Client {
	intervals := newIntervals(cfg)

	// if no retries are configured, that means decoration is disabled
	if len(intervals) == 0 {
		return nil
	}

	if next == nil {
		next = http.DefaultClient
	}

	c := &Client{
		next:           next,
		maxElapsedTime: cfg.MaxElapsedTime,
		intervals:      intervals,
		random:         cfg.Random,
		timer:          cfg.Timer,
		check:          cfg.Check,
	}

	if c.random == nil {
		//nolint:gosec
		c.random = rand.New(
			rand.NewSource(time.Now().Unix()),
		)
	}

	if c.timer == nil {
		c.timer = DefaultTimer
	}

	if c.check == nil {
		c.check = DefaultCheck
	}

	return c
}

// Retries returns the maximum number of retries this Client will attempt.
// This does not include the initial attempt.  This method will never return
// a nonpositive value.
//
// The State instance returned by GetState will reflect this same value.
func (c *Client) Retries() int {
	return c.intervals.Len()
}

// initialize sets up a series of retry attempts for a given request
func (c *Client) initialize(original *http.Request) (s *State, retryCtx context.Context, cancel context.CancelFunc) {
	s = &State{
		retries: c.intervals.Len(),
	}

	retryCtx = withState(original.Context(), s)
	if c.maxElapsedTime > 0 {
		retryCtx, cancel = context.WithTimeout(retryCtx, c.maxElapsedTime)
	} else {
		retryCtx, cancel = context.WithCancel(retryCtx)
	}

	return
}

// Do takes in an original *http.Request and makes an initial attempt plus
// a maximum number of retries in order to satisfy the request.
//
// IMPORTANT: This method can return both a non-nil response and a non-nil error.
// To be consistent with CheckRedirect, in this case the response's Body has already
// been drained and closed.  The *http.Response.Body field will be nil in that case.
func (c *Client) Do(original *http.Request) (*http.Response, error) {
	state, retryCtx, cancel := c.initialize(original)
	defer cancel()

	// make the initial attempt
	response, err := c.next.Do(original.WithContext(retryCtx))
	if !c.check(response, err) {
		// NOTE: leave this response's Body alone, so callers can see it
		return response, err
	}

	// we need a GetBody so that we can replay the body for each retry
	// this is similar to how an http.Client handles 3XX redirects
	getBody := original.GetBody
	if getBody == nil {
		httpaux.Cleanup(response) // consistent with CheckRedirect
		if response != nil {
			response.Body = nil
		}

		return response, &NoGetBodyError{Initial: err}
	}

	for i := 0; i < c.Retries(); i++ {
		wait := c.intervals.duration(c.random, i)
		tc, stop := c.timer(wait)

		// call this here, so that the response is cleaned up
		// while we wait for the next retry
		state.prepareNext(response, err)

		select {
		case <-retryCtx.Done():
			stop()
			return nil, retryCtx.Err()

		case <-tc:
			// continue
		}

		body, err := getBody()
		if err != nil {
			return nil, &GetBodyError{Err: err}
		}

		original.Body = body
		response, err = c.next.Do(original.WithContext(retryCtx))
		if !c.check(response, err) {
			// NOTE: leave this response's Body alone, so callers can see it
			return response, err
		}
	}

	// NOTE: leave this response's Body alone, so callers can see it
	return response, err
}
