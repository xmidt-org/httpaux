/*
Package retry implements retry logic for HTTP clients.

The Client type can be used directly and instantiated with New.

  client := New(Config{
    Retries: 2,
    Interval: 10 * time.Second,
  }, nil) // uses http.DefaultClient

A Client can also be used as client middleware via its Then method.

  client := New(Config{})
  decorated := client.Then(new(http.Client))

Exponential backoff with jitter is also supported.  For example:

  client := New(Config{
    Retries: 2,
    Interval: 10 * time.Second,
    Multiplier: 2.0,
    Jitter: 0.2,
  })

The basic formula for each time to wait before the next retry, in terms of Config fields, is:

  Interval * (Multiplier^n) * randBetween[1-Jitter,1+Jitter]

where n is the 0-based retry.

See the documentation for the Config type for more details.
*/
package retry
