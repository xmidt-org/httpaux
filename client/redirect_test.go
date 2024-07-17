// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RedirectTestSuite struct {
	suite.Suite
}

func (suite *RedirectTestSuite) testCopyHeadersOnRedirectWithNames() {
	var (
		via = []*http.Request{
			&http.Request{
				Header: http.Header{
					"X-Original": []string{"should not be copied"},
				},
			},
			&http.Request{
				Header: http.Header{
					"X-Original":   []string{"overwritten"},
					"Single-Value": []string{"value1"},
					"Multi-Value":  []string{"value1", "value2", "value3"},
					"Empty":        nil, // empty values, so it should be skipped
				},
			},
		}

		request = &http.Request{
			Header: http.Header{
				"Next": []string{"NextValue"},
			},
		}

		copier = CopyHeadersOnRedirect(
			// check that canonicalization is happening ...
			"x-original",
			"sINGLE-VAlue",
			"Multi-Value",
			"Empty",
		)
	)

	suite.NoError(
		copier(request, via),
	)

	suite.Equal(
		http.Header{
			"Next":         []string{"NextValue"}, // should be untouched
			"X-Original":   []string{"overwritten"},
			"Single-Value": []string{"value1"},
			"Multi-Value":  []string{"value1", "value2", "value3"},
		},
		request.Header,
	)
}

func (suite *RedirectTestSuite) testCopyHeadersOnRedirectNoNames() {
	suite.Nil(
		CopyHeadersOnRedirect(),
	)
}

func (suite *RedirectTestSuite) TestCopyHeadersOnRedirect() {
	suite.Run("WithNames", suite.testCopyHeadersOnRedirectWithNames)
	suite.Run("NoNames", suite.testCopyHeadersOnRedirectNoNames)
}

// newVia is a helper function that creates a number of dummy requests
// to use as a via in a check redirect test.
func (suite *RedirectTestSuite) newVia(count int) (via []*http.Request) {
	if count > 0 {
		via = make([]*http.Request, 0, count)
		for len(via) < cap(via) {
			via = append(
				via,
				&http.Request{
					Header: http.Header{},
				},
			)
		}
	}

	return
}

func (suite *RedirectTestSuite) testMaxRedirectsNegative() {
	suite.Error(
		MaxRedirects(-1)(nil, suite.newVia(0)),
	)

	suite.Error(
		MaxRedirects(-5)(nil, suite.newVia(1)),
	)

	suite.Error(
		MaxRedirects(-34871)(nil, suite.newVia(3)),
	)
}

func (suite *RedirectTestSuite) testMaxRedirectsZero() {
	suite.Error(
		MaxRedirects(0)(nil, suite.newVia(0)),
	)

	suite.Error(
		MaxRedirects(0)(nil, suite.newVia(1)),
	)

	suite.Error(
		MaxRedirects(0)(nil, suite.newVia(3)),
	)
}

func (suite *RedirectTestSuite) testMaxRedirectsSuccess() {
	suite.NoError(
		MaxRedirects(1)(nil, suite.newVia(0)),
	)

	suite.NoError(
		MaxRedirects(3)(nil, suite.newVia(1)),
	)

	suite.NoError(
		MaxRedirects(20)(nil, suite.newVia(16)),
	)
}

func (suite *RedirectTestSuite) testMaxRedirectsFail() {
	suite.Error(
		MaxRedirects(1)(nil, suite.newVia(1)),
	)

	suite.Error(
		MaxRedirects(1)(nil, suite.newVia(2)),
	)

	suite.Error(
		MaxRedirects(4)(nil, suite.newVia(4)),
	)

	suite.Error(
		MaxRedirects(3)(nil, suite.newVia(6)),
	)

	suite.Error(
		MaxRedirects(20)(nil, suite.newVia(22)),
	)
}

func (suite *RedirectTestSuite) TestMaxRedirects() {
	suite.Run("Negative", suite.testMaxRedirectsNegative)
	suite.Run("Zero", suite.testMaxRedirectsZero)
	suite.Run("Success", suite.testMaxRedirectsSuccess)
	suite.Run("Fail", suite.testMaxRedirectsFail)
}

func (suite *RedirectTestSuite) checkRedirectSuccess(*http.Request, []*http.Request) error {
	return nil
}

func (suite *RedirectTestSuite) checkRedirectSuccesses(count int) (checks []CheckRedirect) {
	checks = make([]CheckRedirect, 0, count)
	for len(checks) < cap(checks) {
		checks = append(checks, suite.checkRedirectSuccess)
	}

	return
}

func (suite *RedirectTestSuite) checkRedirectFail(*http.Request, []*http.Request) error {
	return errors.New("test error")
}

func (suite *RedirectTestSuite) testNewCheckRedirectsNil() {
	suite.Nil(
		NewCheckRedirects(),
	)

	suite.Nil(
		NewCheckRedirects(nil),
	)

	suite.Nil(
		NewCheckRedirects(nil, nil, nil),
	)
}

func (suite *RedirectTestSuite) testNewCheckRedirectsSuccess() {
	suite.Run("NoNils", func() {
		for _, count := range []int{1, 2, 5} {
			suite.Run(fmt.Sprintf("count=%d", count), func() {
				checkRedirect := NewCheckRedirects(
					suite.checkRedirectSuccesses(count)...,
				)

				suite.Require().NotNil(checkRedirect)

				// won't matter what's passed, as our test functions don't use the parameters
				suite.NoError(checkRedirect(nil, nil))
			})
		}
	})

	suite.Run("WithNils", func() {
		checkRedirect := NewCheckRedirects(
			suite.checkRedirectSuccess,
			nil,
			suite.checkRedirectSuccess,
		)

		suite.Require().NotNil(checkRedirect)

		// won't matter what's passed, as our test functions don't use the parameters
		suite.NoError(checkRedirect(nil, nil))
	})
}

func (suite *RedirectTestSuite) testNewCheckRedirectsFail() {
	suite.Run("NoNils", func() {
		for _, count := range []int{1, 2, 5} {
			suite.Run(fmt.Sprintf("count=%d", count), func() {
				components := suite.checkRedirectSuccesses(count)

				// any fail will fail the entire check
				components[len(components)/2] = suite.checkRedirectFail
				checkRedirect := NewCheckRedirects(components...)

				suite.Require().NotNil(checkRedirect)

				// won't matter what's passed, as our test functions don't use the parameters
				suite.Error(checkRedirect(nil, nil))
			})
		}
	})

	suite.Run("WithNils", func() {
		checkRedirect := NewCheckRedirects(
			suite.checkRedirectFail,
			nil,
			suite.checkRedirectSuccess,
		)

		suite.Require().NotNil(checkRedirect)

		// won't matter what's passed, as our test functions don't use the parameters
		suite.Error(checkRedirect(nil, nil))
	})
}

func (suite *RedirectTestSuite) TestNewCheckRedirects() {
	suite.Run("Nil", suite.testNewCheckRedirectsNil)
	suite.Run("Success", suite.testNewCheckRedirectsSuccess)
	suite.Run("Fail", suite.testNewCheckRedirectsFail)
}

func TestRedirect(t *testing.T) {
	suite.Run(t, new(RedirectTestSuite))
}
