package retry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/httpmock"
)

type ClientTestSuite struct {
	suite.Suite
}

// newRequest creates a standard HTTP request for testing.  Since the request data itself
// shouldn't change between retry attempts, we can just use a canonical request with the
// same state for every test.
func (suite *ClientTestSuite) newRequest() *http.Request {
	r, err := http.NewRequest("GET", "/test", bytes.NewBufferString("test request"))
	suite.Require().NoError(err)
	r.Header.Set("Test", "true")
	return r
}

// mockRoundTripper creates a mock with global assertions appropriate for newRequest.
func (suite *ClientTestSuite) mockRoundTripper() *httpmock.RoundTripper {
	return httpmock.NewRoundTripperSuite(suite).
		AssertRequest(
			httpmock.Methods("GET"),
			httpmock.Path("/test"),
			httpmock.Header("Test", "true"),
		)
}

// mockClient produces a "real" http.Client with a mocked round tripper setup
// via mockRoundTripper.
func (suite *ClientTestSuite) mockClient() (httpaux.Client, *httpmock.RoundTripper) {
	rt := suite.mockRoundTripper()
	return &http.Client{Transport: rt}, rt
}

// assertBody is a helper function for verifying a response body
func (suite *ClientTestSuite) assertBody(expected string, r *http.Response) {
	suite.Require().NotNil(r)
	suite.Require().NotNil(r.Body)

	var actual bytes.Buffer
	_, err := io.Copy(&actual, r.Body)
	suite.Require().NoError(err)
	suite.Equal(expected, actual.String())
}

func (suite *ClientTestSuite) TestNoRetries() {
	suite.Run("New", func() {
		c, _ := suite.mockClient()
		decorated := New(Config{})(c)
		suite.Equal(c, decorated)
	})

	suite.Run("NewClient", func() {
		c, _ := suite.mockClient()
		suite.Nil(NewClient(Config{}, c))
	})
}

func (suite *ClientTestSuite) testDoSuccess(c httpaux.Client, rt *httpmock.RoundTripper, totalRetries, retries int) {
	suite.Require().NotNil(c)
	suite.Require().NotNil(rt)

	for attempt := 0; attempt <= retries; attempt++ {
		attempt := attempt
		rt.OnAny().Return(&http.Response{
			StatusCode: 200,
			Body:       httpmock.Bodyf("attempt %d", attempt),
		}, nil).Run(func(args mock.Arguments) {
			r := args.Get(0).(*http.Request)
			s := GetState(r.Context())
			suite.Require().NotNil(s)
			suite.Equal(attempt, s.Attempt())
			suite.Equal(totalRetries, s.Retries())

			previous, previousErr := s.Previous()
			suite.NoError(previousErr) // we're never returning errors in these tests
			if attempt == 0 {
				suite.Nil(previous)
			} else {
				suite.NotNil(previous)
				suite.Nil(previous.Body)
			}
		}).Once()
	}

	actual, actualErr := c.Do(suite.newRequest())
	suite.NoError(actualErr)
	suite.assertBody(fmt.Sprintf("attempt %d", retries), actual)
	rt.AssertExpectations()
}

func (suite *ClientTestSuite) TestDefaults() {
	suite.Run("New", func() {
		c, rt := suite.mockClient()
		c = New(Config{
			Retries: 1,
		})(c)

		suite.testDoSuccess(c, rt, 1, 0)
	})

	suite.Run("NewClient", func() {
		c, rt := suite.mockClient()
		c = NewClient(Config{
			Retries: 1,
		}, c)

		suite.testDoSuccess(c, rt, 1, 0)
	})
}

func (suite *ClientTestSuite) TestDoSuccess() {
	testData := []struct {
		cfg     Config
		retries int
	}{
		{
			cfg: Config{
				Retries: 1,
			},
			retries: 0,
		},
		{
			cfg: Config{
				Retries: 1,
			},
			retries: 1,
		},
		{
			cfg: Config{
				Retries: 2,
			},
			retries: 1,
		},
		{
			cfg: Config{
				Retries:    2,
				Multiplier: 2.0,
			},
			retries: 1,
		},
		{
			cfg: Config{
				Retries:    3,
				Multiplier: 1.5,
				Jitter:     0.1,
			},
			retries: 2,
		},
		{
			cfg: Config{
				Retries:        5,
				Multiplier:     3.7,
				Jitter:         0.3,
				MaxElapsedTime: 5 * time.Minute,
			},
			retries: 3,
		},
	}

	for i, record := range testData {
		suite.Run(strconv.Itoa(i), func() {
			suite.Run("New", func() {
				v := newVerifier(suite.T(), record.cfg, record.retries)
				record.cfg.Check = v.Check
				record.cfg.Random = v
				record.cfg.Timer = v.Timer

				c, rt := suite.mockClient()
				c = New(record.cfg)(c)
				suite.testDoSuccess(c, rt, record.cfg.Retries, record.retries)
				v.AssertExpectations()
			})

			suite.Run("NewClient", func() {
				v := newVerifier(suite.T(), record.cfg, record.retries)
				record.cfg.Check = v.Check
				record.cfg.Random = v
				record.cfg.Timer = v.Timer

				c, rt := suite.mockClient()
				c = NewClient(record.cfg, c)
				suite.testDoSuccess(c, rt, record.cfg.Retries, record.retries)
				v.AssertExpectations()
			})
		})
	}
}

func (suite *ClientTestSuite) testNoGetBody(c httpaux.Client, rt *httpmock.RoundTripper) {
	body := httpmock.BodyString("first attempt")
	expected := &http.Response{
		Body: body,
	}

	rt.OnAny().Return(expected, nil).Once()

	request := suite.newRequest()
	request.GetBody = nil // force this error
	actual, actualErr := c.Do(request)

	suite.True(expected == actual)
	suite.Nil(actual.Body)
	suite.True(httpmock.Closed(body))
	suite.IsType((*NoGetBodyError)(nil), actualErr)
	suite.NotEmpty(actualErr.Error())
	rt.AssertExpectations()
}

func (suite *ClientTestSuite) TestNoGetBody() {
	cfg := Config{
		Retries: 2,
		Check: func(*http.Response, error) bool {
			return true
		},
		Timer: func(time.Duration) (<-chan time.Time, func() bool) {
			suite.Require().Fail("Timer should not have been called")
			tc := make(chan time.Time)
			close(tc) // avoid deadlocks on failed tests
			return tc, func() bool { return true }
		},
	}

	suite.Run("New", func() {
		c, rt := suite.mockClient()
		c = New(cfg)(c)
		suite.testNoGetBody(c, rt)
	})

	suite.Run("NewClient", func() {
		c, rt := suite.mockClient()
		c = NewClient(cfg, c)
		suite.testNoGetBody(c, rt)
	})
}

func (suite *ClientTestSuite) testGetBodyError(c httpaux.Client, rt *httpmock.RoundTripper) {
	body := httpmock.BodyString("first attempt")
	expected := &http.Response{
		Body: body,
	}

	rt.OnAny().Return(expected, nil).Once()

	getBodyErr := errors.New("expected error from GetBody")
	request := suite.newRequest()
	request.GetBody = func() (io.ReadCloser, error) {
		// force an error
		return nil, getBodyErr
	}
	actual, actualErr := c.Do(request)
	rt.AssertExpectations()

	suite.Nil(actual) // nil response when GetBody returns an error
	suite.True(httpmock.Closed(body))
	suite.IsType((*GetBodyError)(nil), actualErr)
	suite.NotEmpty(actualErr.Error())
	suite.Equal(getBodyErr, actualErr.(*GetBodyError).Err)
}

func (suite *ClientTestSuite) TestGetBodyError() {
	cfg := Config{
		Retries: 2,
		Check: func(*http.Response, error) bool {
			return true
		},
		Timer: func(time.Duration) (<-chan time.Time, func() bool) {
			tc := make(chan time.Time)
			close(tc)
			return tc, func() bool { return true }
		},
	}

	suite.Run("New", func() {
		c, rt := suite.mockClient()
		c = New(cfg)(c)
		suite.testGetBodyError(c, rt)
	})

	suite.Run("NewClient", func() {
		c, rt := suite.mockClient()
		c = NewClient(cfg, c)
		suite.testGetBodyError(c, rt)
	})
}

func (suite *ClientTestSuite) testCanceled(c httpaux.Client, rt *httpmock.RoundTripper) {
	body := httpmock.BodyString("first attempt")
	expected := &http.Response{
		Body: body,
	}

	rt.OnAny().Return(expected, nil).Once()

	request := suite.newRequest()
	ctx, cancel := context.WithCancel(request.Context())
	request = request.WithContext(ctx)

	ready := make(chan struct{})

	type result struct {
		response *http.Response
		err      error
	}

	results := make(chan result)

	go func() {
		close(ready)
		var result result
		result.response, result.err = c.Do(request)
		results <- result
	}()

	select {
	case <-ready:
		// passing
		cancel()
	case <-time.After(2 * time.Second):
		cancel() // cleanup after a failed test
		suite.Require().Fail("Goroutine for client.Do never signaled readiness")
		return
	}

	var r result
	select {
	case r = <-results:
		// passing
	case <-time.After(2 * time.Second):
		suite.Require().Fail("No Do result was returned")
		return
	}

	suite.Nil(r.response)
	suite.True(httpmock.Closed(body))
	suite.True(errors.Is(r.err, context.Canceled))
}

func (suite *ClientTestSuite) TestCanceled() {
	cfg := Config{
		Retries: 1,
		Check: func(*http.Response, error) bool {
			return true
		},
		Timer: func(time.Duration) (<-chan time.Time, func() bool) {
			// this time channel intentionally prevents the timer case from executing
			return make(chan time.Time), func() bool { return true }
		},
	}

	suite.Run("New", func() {
		c, rt := suite.mockClient()
		c = New(cfg)(c)
		suite.testCanceled(c, rt)
	})

	suite.Run("NewClient", func() {
		c, rt := suite.mockClient()
		c = NewClient(cfg, c)
		suite.testCanceled(c, rt)
	})
}

func (suite *ClientTestSuite) TestRetriesExceeded() {
	cfg := Config{
		Retries: 2,
		Check: func(*http.Response, error) bool {
			// unconditionally force all retries, so that our retry count
			// is guaranteed to expire
			return true
		},
		Timer: func(time.Duration) (<-chan time.Time, func() bool) {
			tc := make(chan time.Time)
			close(tc)
			return tc, func() bool { return true }
		},
	}

	suite.Run("New", func() {
		c, rt := suite.mockClient()
		c = New(cfg)(c)
		suite.testDoSuccess(c, rt, cfg.Retries, cfg.Retries)
	})

	suite.Run("NewClient", func() {
		c, rt := suite.mockClient()
		c = NewClient(cfg, c)
		suite.testDoSuccess(c, rt, cfg.Retries, cfg.Retries)
	})
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
