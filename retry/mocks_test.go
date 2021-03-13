package retry

import (
	"net/http"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// verifier is a mocked/stubbed Check, Timer, and Random implementation all in one.
// It verifies that the correct number and type of calls are made, given an expected
// number of retries.
type verifier struct {
	assert  *assert.Assertions
	require *require.Assertions

	// i is the precomputed intervals that the enclosing *Client should be using
	i intervals

	// expectedRetries is the total number of retries the enclosing
	// test will attempt.
	//
	// Check should be called 1 more than this value, since it is called on
	// the last attempt to return false.
	//
	// Timer should be called exactly this many times.
	//
	// Int63n should only be called this many times if Jitter was configured.
	// Otherwise, Int63n should never be called.
	expectedRetries int

	// hasJitter indicates whether Jitter was configured.  If this is false,
	// Int63n should never be called.  If this is true, Int63n should be
	// called 1 less than the expectedRetries.
	hasJitter bool

	checkCount  int
	timerCount  int
	randomCount int
}

func newVerifier(t mock.TestingT, cfg Config, expectedRetries int) *verifier {
	v := &verifier{
		assert:          assert.New(t),
		require:         require.New(t),
		i:               newIntervals(cfg),
		expectedRetries: expectedRetries,
	}

	v.require.LessOrEqual(expectedRetries, len(v.i), "Test is improperly configured")
	v.hasJitter = (cfg.Jitter > 0.0 && cfg.Jitter < 1.0)
	return v
}

// AssertExpectations asserts that the correct number of each kind of call was made
func (v *verifier) AssertExpectations() {
	v.assert.Equal(v.expectedRetries+1, v.checkCount, "Incorrect number of Check calls")
	v.assert.Equal(v.expectedRetries, v.timerCount, "Incorrect number of Timer calls")
	if v.hasJitter {
		v.assert.Equal(v.expectedRetries, v.randomCount, "Incorrect number of Random calls")
	} else {
		v.assert.Zero(v.randomCount, "Random should never be called when there is no Jitter")
	}
}

// Check is a Config.Check strategy that returns false until the expected number
// of retries is met.
func (v *verifier) Check(*http.Response, error) bool {
	v.checkCount++
	return v.checkCount <= v.expectedRetries
}

// AlwaysCheck is a Config.Check that always returns true, but still tracks
// number of calls for AssertExpectations.
func (v *verifier) AlwaysCheck(response *http.Response, err error) bool {
	v.Check(response, err)
	return true
}

// Int63n implements Random and always returns 0 so that timer durations
// are predictable.
func (v *verifier) Int63n(jitter int64) int64 {
	v.randomCount++
	return 0
}

// Timer is a Config.Timer implementation that always returns a closed time channel
// and a noop stop function.  This allows retries to continue immediately.
func (v *verifier) Timer(base time.Duration) (<-chan time.Time, func() bool) {
	v.timerCount++

	tc := make(chan time.Time)
	close(tc)
	return tc, func() bool { return true }
}
