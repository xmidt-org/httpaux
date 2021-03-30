package retry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/erraux"
)

type CheckTestSuite struct {
	suite.Suite
}

func (suite *CheckTestSuite) TestDefaultCheck() {
	testData := []struct {
		response *http.Response
		err      error
		expected bool
	}{
		{
			response: nil,
			err:      nil,
			expected: false,
		},
		{
			response: &http.Response{
				StatusCode: 200,
			},
			err:      nil,
			expected: false, // this was successful ... no need to retry
		},
		{
			response: &http.Response{
				StatusCode: http.StatusRequestTimeout,
			},
			err:      nil,
			expected: true,
		},
		{
			response: &http.Response{
				StatusCode: http.StatusTooManyRequests,
			},
			err:      nil,
			expected: true,
		},
		{
			response: &http.Response{
				StatusCode: http.StatusGatewayTimeout,
			},
			err:      nil,
			expected: true,
		},
		{
			response: &http.Response{
				StatusCode: 200,
			},
			err: &net.DNSError{IsTemporary: true},
			// both response and error are not nil, but the error indicates a retry
			expected: true,
		},
		{
			response: nil,
			err:      errors.New("this shouldn't be retried"),
			expected: false,
		},
		{
			response: nil,
			err:      &erraux.Error{Err: errors.New("this wrapped error shouldn't be retried")},
			expected: false,
		},
		{
			response: nil,
			err:      context.Canceled,
			expected: false,
		},
		{
			response: nil,
			err:      &erraux.Error{Err: context.Canceled},
			expected: false,
		},
		{
			response: nil,
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			response: nil,
			err:      &erraux.Error{Err: context.DeadlineExceeded},
			expected: false,
		},
		{
			response: nil,
			err:      &net.DNSError{IsTemporary: true},
			expected: true,
		},
		{
			response: nil,
			err:      &erraux.Error{Err: &net.DNSError{IsTemporary: true}},
			expected: true,
		},
		{
			response: nil,
			err:      &net.DNSError{IsTemporary: false},
			expected: false,
		},
		{
			response: nil,
			err:      &erraux.Error{Err: &net.DNSError{IsTemporary: false}},
			expected: false,
		},
	}

	for i, record := range testData {
		suite.Run(strconv.Itoa(i), func() {
			suite.Equal(
				record.expected,
				DefaultCheck(record.response, record.err),
			)
		})
	}
}

func TestCheck(t *testing.T) {
	suite.Run(t, new(CheckTestSuite))
}
