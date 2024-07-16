// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package retry

import (
	"context"
	"net/http"

	"github.com/xmidt-org/httpaux"
)

// State is the current state of a retry operation.  Instances
// of this type are available in request context's during the initial
// attempt and any retries.
//
// This type is never safe for concurrent access.
type State struct {
	attempt int
	retries int

	previous    *http.Response
	previousErr error
}

// Attempt is the 0-based attempt to execute this HTTP transaction.
// Zero (0) means the initial attempt.  Positive values indicate
// the retry attempt.  In other words, this method returns the
// number of attempts that have been tried previously.
func (s *State) Attempt() int {
	return s.attempt
}

// Retries is the maximum number of retries that will be attempted
// Attempt will always return a number less than or equal to this value.
// The Retries value will never change during a sequence of attempts.
func (s *State) Retries() int {
	return s.retries
}

// Previous is the result of the previous attempt.  The response body
// will not be available, but other information such as headers and
// the status code will be.
//
// This method will return (nil, nil) for the first attempt, i.e.
// when Attempt() returns 0.
func (s *State) Previous() (*http.Response, error) {
	return s.previous, s.previousErr
}

// prepareNext preps for the next attempt in a series.
// the given response's body is drained, closed, and nil'd out.
func (s *State) prepareNext(previous *http.Response, previousErr error) {
	s.attempt++
	httpaux.Cleanup(previous)
	if previous != nil {
		previous.Body = nil
	}

	s.previous = previous
	s.previousErr = previousErr
}

// contextKey is the internal context.Context key that stores the *State
type contextKey struct{}

// GetState returns the retry State associated with the given context.
// Decorated code can make use of this for metrics, logging, etc.
//
// IMPORTANT: State is not safe for concurrent access.  The State instance
// returned by this function should never be retained.
func GetState(ctx context.Context) *State {
	s, _ := ctx.Value(contextKey{}).(*State)
	return s
}

// withState creates a subcontext that stores the given state
func withState(ctx context.Context, s *State) context.Context {
	return context.WithValue(ctx, contextKey{}, s)
}
