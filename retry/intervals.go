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

	i := make(intervals, cfg.Retries)

	// "seed" the first element
	i[0].base = cfg.Interval
	if i[0].base <= 0 {
		i[0].base = DefaultInterval
	}

	// first pass: apply the multiplier cumulatively.
	// this is where we get the exponential backoff
	m := cfg.Multiplier
	if m <= 0.0 {
		m = 1.0
	}

	for x := 1; x < i.Len(); x++ {
		i[x].base = time.Duration(
			math.Round(float64(i[x-1].base) * m),
		)
	}

	// second pass: if applicable, apply jitter
	// the jitter window will change if the multiplier was not 1.0

	if cfg.Jitter > 0.0 && cfg.Jitter < 1.0 {
		jitterLo, jitterHi := 1-cfg.Jitter, 1+cfg.Jitter

		for x := 0; x < i.Len(); x++ {
			// compute the limits of the jitter window
			lo := int64(
				math.Round(float64(i[x].base) * jitterLo),
			)

			hi := int64(
				math.Round(float64(i[x].base) * jitterHi),
			)

			// this precomputation allows the actual retry wait time
			// to be computed simply:
			//
			// base + rand.Int63n(jitter)
			//
			// we add one to the jitter because rand.Int63n returns a value
			// in the range [0,n)
			i[x].base = time.Duration(lo)
			i[x].jitter = hi - lo + 1
		}
	}

	return i
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
