package retry

import (
	"math"
	"math/rand"
	"time"
)

// Random is the subset of rand.Rand methods used by this package
// to compute jitter.  *rand.Rand implements this interface.
type Random interface {
	Int63n(int64) int64
}

var _ Random = (*rand.Rand)(nil)

// interval is a precomputed retry interval
type interval struct {
	// base is the base duration to wait until retrying this attempt
	base time.Duration

	// jitter is the range of random values to add to base.
	// if 0, then no jitter is computed.
	jitter int64
}

// duration computes the time to wait before attempting this retry
func (i interval) duration(r Random) time.Duration {
	if i.jitter > 0 {
		return i.base + time.Duration(r.Int63n(i.jitter))
	}

	return i.base
}

// intervals is a sequence of precomputed retry interval tuples
type intervals []interval

// newIntervals precomputes retry intervals given a Config.
// If cfg.Retries is nonpositive, this function returns an empty slice.
func newIntervals(cfg Config) intervals {
	if cfg.Retries < 1 {
		return nil // no retries
	}

	intervals := make([]interval, cfg.Retries)
	if cfg.Interval > 0 {
		intervals[0].base = cfg.Interval
	} else {
		intervals[0].base = DefaultInterval
	}

	// first pass: apply the multiplier to each interval
	for i := 1; i < len(intervals); i++ {
		if cfg.Multiplier > 0.0 && cfg.Multiplier != 1.0 {
			intervals[i].base = time.Duration(
				math.Round(float64(intervals[i-1].base) * cfg.Multiplier),
			)
		} else {
			intervals[i].base = intervals[i-1].base
		}
	}

	// second pass: precompute the jitter ranges
	if cfg.Jitter > 0.0 && cfg.Jitter < 1.0 {
		jitterLo, jitterHi := 1-cfg.Jitter, 1+cfg.Jitter
		for i := 0; i < len(intervals); i++ {
			base := intervals[i].base

			// readjust to the low end of a range
			intervals[i].base = time.Duration(
				math.Round(float64(base) * jitterLo),
			)

			// now compute the integral range of values
			limit := int64(
				math.Round(float64(base) * jitterHi),
			)

			// add one since we use rand.Int63n and we want the highest value included
			intervals[i].jitter = limit + 1
		}
	}

	return intervals
}

// Len returns the count of precomputed intervals.  This is the same
// as the number of retries that will be attempted, e.g. Config.Retries.
func (i intervals) Len() int {
	return len(i)
}

// duration computes the time to wait before the given attempt
// no bounds checking is done by this method.  it will panic if
// attempt is out of bounds.
func (i intervals) duration(r Random, attempt int) time.Duration {
	return i[attempt].duration(r)
}
