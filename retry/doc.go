/*
Package retry implements retry logic for HTTP clients.

The Client type can be used directly and instantiated with NewClient.
This function returns nil if the Config is configured for no retries.

The preferred way to use this package is by creating a middleware constructor
with New:

  ctor := New(Config{
    Retries: 2,
    Interval: 10 * time.Second,
  })

  c := &http.Client{
    Transport: ctor(new(http.Transport)), // can also pass nil to take the default transport
  }

Exponential backoff with jitter is also supported.  For example:

  ctor := New(Config{
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
