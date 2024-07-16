// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package retry

import (
	"context"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/xmidt-org/httpaux"
)

// GetBodyError indicates that http.Request.GetBody returned an error.  Retries
// cannot continue in this case, since the original request body is unavailable.
//
// A request may not have GetBody set.  In that case, a Client will not attempt
// to refresh the body before each retry.
type GetBodyError struct {
	// Err is the error returned from GetBody
	Err error
}

// Unwrap returns the actual error returned from GetBody.
func (err *GetBodyError) Unwrap() error {
	return err.Err
}

// Error fulfills the error interface.
func (err *GetBodyError) Error() string {
	var o strings.Builder
	o.WriteString("GetBody returned an error: [")
	o.WriteString(err.Err.Error())
	o.WriteRune(']')
	return o.String()
}

// Client is an httpaux.Client that retries HTTP transactions.  Instances can be used
// in (2) ways:
//
// (1) Client.Do can execute HTTP transactions using whatever retry semantics were
// defined in New.  If Config.Retries was nonpositive, the Client won't perform any
// retries.
//
// (2) Client.Then can be used as client middleware to attach retry semantics
// to other HTTP clients.
type Client struct {
	// next is the decorated client used to execute HTTP transactions
	next httpaux.Client

	// intervals is the precomputed retry intervals, including jitter
	intervals intervals

	// random is the source of randomness
	random Random

	// timer is the Timer strategy used to wait between retries
	timer Timer

	// check is the Check strategy for determining if a request should be retried
	check Check
}

// New constructs a Client from a configuration.  If cfg.Retries
// is nonpositive, the returned client will not do retries.
//
// The next instance is used to actually execute HTTP transactions.  If next
// is nil, http.DefaultClient is used.
func New(cfg Config, next httpaux.Client) (c *Client) {
	c = &Client{
		next:      next,
		intervals: newIntervals(cfg),
		random:    cfg.Random,
		timer:     cfg.Timer,
		check:     cfg.Check,
	}

	if c.next == nil {
		c.next = http.DefaultClient
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

	return
}

// Retries returns the maximum number of retries this Client will attempt.
// This does not include the initial attempt.  If nonpositive, then no retries
// will be attempted.
//
// The State instance returned by GetState will reflect this same value.
func (c *Client) Retries() int {
	return c.intervals.Len()
}

// initialize sets up a series of retry attempts for a given request
func (c *Client) initialize(original *http.Request) (*State, context.Context) {
	s := &State{
		retries: c.intervals.Len(),
	}

	retryCtx := withState(original.Context(), s)
	return s, retryCtx
}

// Do takes in an original *http.Request and makes an initial attempt plus
// a maximum number of retries in order to satisfy the request.
//
// For each request, a State instance is put into the request's context that
// allows decorated clients to access information about the current attempt.
// Decorated code can obtain this State with a call to GetState.
func (c *Client) Do(original *http.Request) (*http.Response, error) {
	state, retryCtx := c.initialize(original)

	// make the initial attempt
	// if no retries were configured, we won't even bother to invoke the check
	response, err := c.next.Do(original.WithContext(retryCtx))
	if len(c.intervals) == 0 || !c.check(response, err) {
		// NOTE: leave this response's Body alone, so callers can see it
		return response, err
	}

	getBody := original.GetBody
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

		if getBody != nil {
			body, err := getBody()
			if err != nil {
				return nil, &GetBodyError{Err: err}
			}

			original.Body = body
		}

		response, err = c.next.Do(original.WithContext(retryCtx))
		if retryCtx.Err() != nil {
			httpaux.Cleanup(response) // just in case we have a misbehaving next client
			return nil, retryCtx.Err()
		} else if !c.check(response, err) {
			// NOTE: leave this response's Body alone, so callers can see it
			return response, err
		}
	}

	// NOTE: leave this response's Body alone, so callers can see it
	return response, err
}

// Then is a client middleware that decorates another client with this
// instance's retry semantics.  If next is nil, http.DefaultClient is decorated.
func (c *Client) Then(next httpaux.Client) httpaux.Client {
	clone := new(Client)
	*clone = *c
	clone.next = next
	if clone.next == nil {
		clone.next = http.DefaultClient
	}

	return clone
}
