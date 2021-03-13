//nolint:bodyclose // none of these test use a "real" response body
package retry

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/httpmock"
)

type ClientTestSuite struct {
	suite.Suite

	transport *http.Transport
	server    *httptest.Server
	serverURL string

	// expected is a request that the handler should compare against
	expected *http.Request

	// expectedBody is the next request body the handler should expect
	expectedBody string
}

var _ suite.SetupAllSuite = (*ClientTestSuite)(nil)
var _ suite.TearDownAllSuite = (*ClientTestSuite)(nil)

func (suite *ClientTestSuite) SetupSuite() {
	suite.transport = new(http.Transport)
	suite.server = httptest.NewServer(
		http.HandlerFunc(suite.handler),
	)

	suite.serverURL = suite.server.URL + "/test"
}

// stateAsserter is a factory for a RequestAsserter that checks the contextual State.
func (suite *ClientTestSuite) stateAsserter(attempt, retries int) httpmock.RequestAsserterFunc {
	return func(a *assert.Assertions, r *http.Request) {
		s := GetState(r.Context())
		suite.Require().NotNil(s)

		suite.Equal(attempt, s.Attempt())
		suite.Equal(retries, s.Retries())

		previous, err := s.Previous()
		suite.NoError(err) // no tests return errors
		if attempt == 0 {
			suite.Nil(previous)
		} else {
			suite.Require().NotNil(previous)
			suite.Nil(previous.Body)
		}
	}
}

// mockSequence mocks out the sequence of calls that will be made to a round tripper given
// an expected number of retries.  the total retries are used to assert that the State is correct.
func (suite *ClientTestSuite) mockSequence(rt *httpmock.RoundTripper, expected, total int) {
	// each attempt, which will always have at least 1 try which is the initial try
	for x := 0; x <= expected; x++ {
		rt.OnAny().AssertRequest(suite.stateAsserter(x, total)).Once()
	}
}

func (suite *ClientTestSuite) TearDownSuite() {
	suite.server.Close()
	suite.transport.CloseIdleConnections()
}

// handler is the HTTP handler that receives our requests.  This method writes a
// known response and verifies the request against the expected fields.
func (suite *ClientTestSuite) handler(rw http.ResponseWriter, actual *http.Request) {
	suite.Equal(suite.expected.Method, actual.Method)
	suite.Equal(suite.expected.URL.Path, actual.URL.Path)
	suite.Equal(int64(len(suite.expectedBody)), actual.ContentLength)

	b, err := ioutil.ReadAll(actual.Body)
	suite.NoError(err)
	suite.Equal(suite.expectedBody, string(b))

	// add some known items to the response
	rw.Header().Set("ClientTestSuite", "true")
	rw.WriteHeader(299)
	rw.Write([]byte("ClientTestSuite"))
}

// assertResponse verifies that the response came through from our handler
// reasonably unchanged.
func (suite *ClientTestSuite) assertResponse(r *http.Response) {
	suite.Require().NotNil(r)
	suite.Equal(299, r.StatusCode)
	suite.Equal("true", r.Header.Get("ClientTestSuite"))

	suite.Require().NotNil(r.Body)
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	suite.NoError(err)
	suite.Equal("ClientTestSuite", string(b))
}

// setupRequest initializes a new request for the handler method to verify.
func (suite *ClientTestSuite) setupRequest(method string, body string) *http.Request {
	var (
		r   *http.Request
		err error
	)

	if len(body) > 0 {
		r, err = http.NewRequest(method, suite.serverURL+"/test", bytes.NewBufferString(body))
	} else {
		r, err = http.NewRequest(method, suite.serverURL+"/test", nil)
	}

	suite.Require().NoError(err)
	suite.Require().NotNil(r)

	suite.expected = r.WithContext(context.Background()) // make a safe clone
	suite.expectedBody = body

	return r
}

// checkNeverCalled is a Check that asserts that it shouldn't be called, as
// in 0 retries were configured.
func (suite *ClientTestSuite) checkNeverCalled(*http.Response, error) bool {
	suite.Fail("The Check strategy should never have been called")
	return false
}

// timerNeverCalled is a Timer that asserts that it shouldn't be called, which is
// the case when 0 retries were configured.
func (suite *ClientTestSuite) timerNeverCalled(time.Duration) (<-chan time.Time, func() bool) {
	suite.Fail("The Timer strategy should never have been called")

	// create a closed timer channel to prevent failing tests from deadlocking
	c := make(chan time.Time)
	close(c)
	return c, func() bool { return true }
}

// randomNeverCalled is a Random that asserts that it shouldn't be called, which is
// the case when 0 retries were configured.
func (suite *ClientTestSuite) randomNeverCalled(int64) int64 {
	suite.Fail("The Random strategy should never have been called")
	return 0
}

func (suite *ClientTestSuite) TestDefaults() {
	suite.T().Logf("Using a zero value for Config and nil for the decorated client should be valid")
	suite.Run("Do", func() {
		var (
			client = New(Config{}, nil)

			requestCtx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
			expected           = suite.setupRequest("GET", "").WithContext(requestCtx)
		)

		defer cancel()
		suite.Zero(client.Retries())

		response, err := client.Do(expected)
		suite.NoError(err)
		suite.assertResponse(response)
	})

	suite.Run("Then", func() {
		var (
			client = New(Config{}, nil).Then(nil)

			requestCtx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
			expected           = suite.setupRequest("GET", "").WithContext(requestCtx)
		)

		defer cancel()

		response, err := client.Do(expected)
		suite.NoError(err)
		suite.assertResponse(response)
	})
}

func (suite *ClientTestSuite) TestNoRetries() {
	suite.T().Logf("No retries are configured.  Only (1) attempt should be made.")
	testCases := []struct {
		cfg    Config
		method string
		body   string
	}{
		{
			cfg:    Config{},
			method: "DELETE",
			body:   "",
		},
		{
			cfg:    Config{},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
		{
			cfg: Config{
				Retries: -1, // should still be no retries
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				Retries: -1, // should still be no retries
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
		{
			cfg: Config{
				MaxElapsedTime: 10 * time.Second,
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				MaxElapsedTime: 10 * time.Second,
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
		{
			cfg: Config{
				Interval:   5 * time.Second,
				Jitter:     0.2,
				Multiplier: 2.5,
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				Interval:   5 * time.Second,
				Jitter:     0.2,
				Multiplier: 2.5,
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
	}

	for i, testCase := range testCases {
		testCase.cfg.Check = suite.checkNeverCalled
		testCase.cfg.Timer = suite.timerNeverCalled
		testCase.cfg.Random = RandomFunc(suite.randomNeverCalled)

		suite.Run(strconv.Itoa(i), func() {
			suite.Run("Do", func() {
				var (
					rt     = httpmock.NewRoundTripperSuite(suite).Next(suite.transport)
					client = New(testCase.cfg, &http.Client{
						Transport: rt,
					})
				)

				suite.Require().Zero(client.Retries())
				suite.mockSequence(rt, 0, 0)
				response, err := client.Do(suite.setupRequest(testCase.method, testCase.body))
				suite.NoError(err)
				suite.assertResponse(response)

				rt.AssertExpectations()
			})

			suite.Run("Then", func() {
				var (
					rt     = httpmock.NewRoundTripperSuite(suite).Next(suite.transport)
					client = New(testCase.cfg, nil).Then(&http.Client{
						Transport: rt,
					})
				)

				suite.mockSequence(rt, 0, 0)
				response, err := client.Do(suite.setupRequest(testCase.method, testCase.body))
				suite.NoError(err)
				suite.assertResponse(response)

				rt.AssertExpectations()
			})
		})
	}
}

func (suite *ClientTestSuite) TestRetries() {
	suite.T().Logf("Retries execute and properly terminate.")
	testCases := []struct {
		cfg    Config
		method string
		body   string
	}{
		{
			cfg: Config{
				Retries: 2,
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				Retries: 3,
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
		{
			cfg: Config{
				Retries:        3,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				Retries:        3,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
		{
			cfg: Config{
				Retries:        3,
				Interval:       17 * time.Second,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				Retries:        3,
				Interval:       17 * time.Second,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
		{
			cfg: Config{
				Retries:        3,
				Interval:       15 * time.Minute,
				Multiplier:     1.75,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				Retries:        3,
				Interval:       15 * time.Minute,
				Multiplier:     1.75,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
		{
			cfg: Config{
				Retries:        3,
				Interval:       22 * time.Second,
				Jitter:         0.2,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				Retries:        3,
				Interval:       22 * time.Second,
				Jitter:         0.2,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
		{
			cfg: Config{
				Retries:        3,
				Interval:       2 * time.Minute,
				Multiplier:     2.5,
				Jitter:         0.3,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "DELETE",
			body:   "",
		},
		{
			cfg: Config{
				Retries:        3,
				Interval:       2 * time.Minute,
				Multiplier:     2.5,
				Jitter:         0.3,
				MaxElapsedTime: 10 * time.Second,
			},
			method: "POST",
			body:   "a lovely POST'ed body",
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			suite.Run("AlwaysCheck", func() {
				suite.T().Logf("The Check strategy never returns false, forcing the retries to elapse")
				suite.Run("Do", func() {
					cfg := testCase.cfg
					v := newVerifier(suite.T(), cfg, cfg.Retries)
					cfg.Check = v.AlwaysCheck
					cfg.Timer = v.Timer
					cfg.Random = v

					var (
						rt = httpmock.NewRoundTripperSuite(suite).Next(suite.transport)

						client = New(cfg, &http.Client{
							Transport: rt,
						})
					)

					suite.Equal(cfg.Retries, client.Retries())
					suite.mockSequence(rt, cfg.Retries, cfg.Retries)
					response, err := client.Do(suite.setupRequest(testCase.method, testCase.body))
					suite.NoError(err)
					suite.assertResponse(response)

					rt.AssertExpectations()
					v.AssertExpectations()
				})

				suite.Run("Then", func() {
					cfg := testCase.cfg
					v := newVerifier(suite.T(), cfg, cfg.Retries)
					cfg.Check = v.AlwaysCheck
					cfg.Timer = v.Timer
					cfg.Random = v

					var (
						rt = httpmock.NewRoundTripperSuite(suite).Next(suite.transport)

						client = New(cfg, nil).Then(&http.Client{
							Transport: rt,
						})
					)

					suite.mockSequence(rt, cfg.Retries, cfg.Retries)
					response, err := client.Do(suite.setupRequest(testCase.method, testCase.body))
					suite.NoError(err)
					suite.assertResponse(response)

					rt.AssertExpectations()
					v.AssertExpectations()
				})
			})

			suite.Run("OneRetry", func() {
				suite.T().Logf("The initial attempt fails, but the first retry succeeds.")
				suite.Run("Do", func() {
					cfg := testCase.cfg
					v := newVerifier(suite.T(), cfg, 1)
					cfg.Check = v.Check
					cfg.Timer = v.Timer
					cfg.Random = v

					var (
						rt = httpmock.NewRoundTripperSuite(suite).Next(suite.transport)

						client = New(cfg, &http.Client{
							Transport: rt,
						})
					)

					suite.Equal(testCase.cfg.Retries, client.Retries())
					suite.mockSequence(rt, 1, cfg.Retries)
					response, err := client.Do(suite.setupRequest(testCase.method, testCase.body))
					suite.NoError(err)
					suite.assertResponse(response)

					rt.AssertExpectations()
					v.AssertExpectations()
				})

				suite.Run("Then", func() {
					cfg := testCase.cfg
					v := newVerifier(suite.T(), cfg, 1)
					cfg.Check = v.Check
					cfg.Timer = v.Timer
					cfg.Random = v

					var (
						rt = httpmock.NewRoundTripperSuite(suite).Next(suite.transport)

						client = New(cfg, nil).Then(&http.Client{
							Transport: rt,
						})
					)

					suite.mockSequence(rt, 1, cfg.Retries)
					response, err := client.Do(suite.setupRequest(testCase.method, testCase.body))
					suite.NoError(err)
					suite.assertResponse(response)

					rt.AssertExpectations()
					v.AssertExpectations()
				})
			})
		})
	}
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
