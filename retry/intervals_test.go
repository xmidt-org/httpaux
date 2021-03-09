package retry

import (
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockRandom struct {
	mock.Mock
}

func (m *mockRandom) Int63n(v int64) int64 {
	arguments := m.Called(v)
	return arguments.Get(0).(int64)
}

type IntervalsTestSuite struct {
	suite.Suite
}

func (suite *IntervalsTestSuite) TestNoRetries() {
	i := newIntervals(Config{})
	suite.Empty(i)
	suite.Zero(i.Len())
}

func (suite *IntervalsTestSuite) testDurationSimple() {
	testData := []struct {
		cfg      Config
		expected intervals
	}{
		{
			cfg: Config{
				Retries: 1,
			},
			expected: intervals{
				{base: DefaultInterval},
			},
		},
		{
			cfg: Config{
				Retries:  2,
				Interval: 27 * time.Minute,
			},
			expected: intervals{
				{base: 27 * time.Minute}, {base: 27 * time.Minute},
			},
		},
		{
			cfg: Config{
				Retries:  5,
				Interval: 13 * time.Second,
			},
			expected: intervals{
				{base: 13 * time.Second}, {base: 13 * time.Second}, {base: 13 * time.Second}, {base: 13 * time.Second}, {base: 13 * time.Second},
			},
		},
	}

	for i, record := range testData {
		suite.Run(strconv.Itoa(i), func() {
			actual := newIntervals(record.cfg)
			suite.Equal(record.expected, actual)

			for attempt := 0; attempt < actual.Len(); attempt++ {
				suite.Run(fmt.Sprintf("attempt:%d", attempt), func() {
					m := new(mockRandom)
					suite.Equal(actual[attempt].base, actual.duration(m, attempt))
					m.AssertExpectations(suite.T())
				})
			}
		})
	}
}

func (suite *IntervalsTestSuite) testDurationMultiplier() {
	testData := []struct {
		cfg      Config
		expected intervals
	}{
		{
			cfg: Config{
				Retries:    1,
				Multiplier: 1.2,
			},
			expected: intervals{
				{base: DefaultInterval}, // multiplier only applies to subsequent attempts
			},
		},
		{
			cfg: Config{
				Retries:    2,
				Interval:   20 * time.Minute,
				Multiplier: 1.5,
			},
			expected: intervals{
				{base: 20 * time.Minute}, {base: 30 * time.Minute},
			},
		},
		{
			cfg: Config{
				Retries:    2,
				Interval:   20 * time.Minute,
				Multiplier: -1.5, // will be ignored
			},
			expected: intervals{
				{base: 20 * time.Minute}, {base: 20 * time.Minute},
			},
		},
		{
			cfg: Config{
				Retries:    2,
				Interval:   20 * time.Minute,
				Multiplier: 1.0, // same as unset
			},
			expected: intervals{
				{base: 20 * time.Minute}, {base: 20 * time.Minute},
			},
		},
		{
			cfg: Config{
				Retries:    5,
				Interval:   1 * time.Minute,
				Multiplier: 2.0,
			},
			expected: intervals{
				{base: 1 * time.Minute}, {base: 2 * time.Minute}, {base: 4 * time.Minute}, {base: 8 * time.Minute}, {base: 16 * time.Minute},
			},
		},
	}

	for i, record := range testData {
		suite.Run(strconv.Itoa(i), func() {
			actual := newIntervals(record.cfg)
			suite.Equal(record.expected, actual)

			for attempt := 0; attempt < actual.Len(); attempt++ {
				suite.Run(fmt.Sprintf("attempt:%d", attempt), func() {
					m := new(mockRandom)
					suite.Equal(actual[attempt].base, actual.duration(m, attempt))
					m.AssertExpectations(suite.T())
				})
			}
		})
	}
}

func (suite *IntervalsTestSuite) testDurationJitter() {
	testData := []struct {
		cfg      Config
		expected intervals
	}{
		{
			cfg: Config{
				Retries:    1,
				Multiplier: 0.5, // won't matter, since only 1 retry
				Jitter:     0.2,
			},
			expected: intervals{
				{
					// don't precompute for tests here, so we can change DefaultInterval easily
					base: time.Duration(math.Round(float64(DefaultInterval) * (1 - 0.2))),

					// the range of numbers from 1-jitter to 1+jitter, plus one
					jitter: int64(math.Round(float64(DefaultInterval)*(1+0.2))) -
						int64(math.Round(float64(DefaultInterval)*(1-0.2))) + 1,
				},
			},
		},
		{
			cfg: Config{
				Retries:  2,
				Interval: 10 * time.Second,
				Jitter:   0.3,
			},
			expected: intervals{
				// no multiplier, so everything's the same
				{base: 7 * time.Second, jitter: int64(6*time.Second) + 1},
				{base: 7 * time.Second, jitter: int64(6*time.Second) + 1},
			},
		},
		{
			cfg: Config{
				Retries:    2,
				Interval:   10 * time.Second,
				Multiplier: 2.0,
				Jitter:     0.3,
			},
			expected: intervals{
				{base: 7 * time.Second, jitter: int64(6*time.Second) + 1},
				{base: 14 * time.Second, jitter: int64(12*time.Second) + 1},
			},
		},
		{
			cfg: Config{
				Retries:  5,
				Interval: 5 * time.Minute,
				Jitter:   0.8,
			},
			expected: intervals{
				// no multiplier, so everything's the same
				{base: 60 * time.Second, jitter: int64(480*time.Second) + 1},
				{base: 60 * time.Second, jitter: int64(480*time.Second) + 1},
				{base: 60 * time.Second, jitter: int64(480*time.Second) + 1},
				{base: 60 * time.Second, jitter: int64(480*time.Second) + 1},
				{base: 60 * time.Second, jitter: int64(480*time.Second) + 1},
			},
		},
		{
			cfg: Config{
				Retries:    5,
				Interval:   5 * time.Minute,
				Multiplier: 2.5,
				Jitter:     0.8,
			},
			expected: intervals{
				{base: 60 * time.Second, jitter: int64(480*time.Second) + 1},
				{base: 150 * time.Second, jitter: int64(1200*time.Second) + 1},
				{base: 375 * time.Second, jitter: int64(3000*time.Second) + 1},
				{base: 9375 * time.Second / 10, jitter: int64(7500*time.Second) + 1},
				{base: 234375 * time.Second / 100, jitter: int64(18750*time.Second) + 1},
			},
		},
	}

	for i, record := range testData {
		suite.Run(strconv.Itoa(i), func() {
			actual := newIntervals(record.cfg)
			suite.Equal(record.expected, actual)

			for attempt := 0; attempt < actual.Len(); attempt++ {
				suite.Run(fmt.Sprintf("attempt:%d", attempt), func() {
					m := new(mockRandom)
					m.On("Int63n", actual[attempt].jitter).Return(int64(0)).Once()
					suite.Equal(actual[attempt].base, actual.duration(m, attempt))
					m.AssertExpectations(suite.T())
				})
			}
		})
	}
}

func (suite *IntervalsTestSuite) TestDuration() {
	suite.Run("Simple", suite.testDurationSimple)
	suite.Run("Multiplier", suite.testDurationMultiplier)
	suite.Run("Jitter", suite.testDurationJitter)
}

func TestIntervals(t *testing.T) {
	suite.Run(t, new(IntervalsTestSuite))
}
