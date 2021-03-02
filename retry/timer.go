package retry

import "time"

// Timer is a strategy for starting a timer with its stop function
type Timer func(time.Duration) (<-chan time.Time, func() bool)

// DefaultTimer is the default Timer implementation.  It simply
// delegates to time.NewTimer.
func DefaultTimer(d time.Duration) (<-chan time.Time, func() bool) {
	t := time.NewTimer(d)
	return t.C, t.Stop
}
