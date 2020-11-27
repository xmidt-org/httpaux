/*
Package roundtrip provides middleware and a few utility types for use with
http.RoundTripper

Middleware

The Constructor and Chain types provide a way for application code to decorate
an http.RoundTripper, usually an http.Transport, to any arbitrary depth.  These
types are analogous to https://pkg.go.dev/github.com/justinas/alice.

This package provide the Func type as an analog to http.HandlerFunc.  When implementing
middleware, chaining function calls through Func works as with serverside middleware.

CloseIdleConnections

When decorating http.RoundTripper, care should be take to avoid covering up the
CloseIdleConnections method.  This method is used by the enclosing http.Client.
If middleware does not properly expose CloseIdleConnections, then there is no way
to invoke this method through the client:

  // CloseIdleConnections is lost in decoration
  func MyConstructor(next http.RoundTripper) http.RoundTripper {
    return roundtrip.Func(func(request *http.Request) (*http.Response, error) {
      // some interesting decorator code here
      return next.RoundTrip(request)
    })
  }

To assist with this, PreserveCloseIdler is provided to ensure that Constructor code
properly exposes a CloseIdleConnections method:

  // CloseIdleConnections is now available on the returned http.RoundTripper
  func MyConstructor(next http.RoundTripper) http.RoundTripper {
    return roundtrip.PreserveCloseIdler(
      next,
      roundtrip.Func(func(request *http.Request) (*http.Response, error) {
        // some interesting decorator code here
        return next.RoundTrip(request)
      }),
    )
  }

If a constructor wants to decorate both the RoundTrip and CloseIdleConnections methods,
the Decorator type and CloseIdleConnections function in this package can be used to facilitate this:

  func MyConstructor(next http.RoundTripper) http.RoundTripper {
    return roundtrip.Decorator{
      RoundTrip: roundTrip.Func(func(request *http.Request) (*http.Response, error) {
        // some interesting decorator code here
        return next.RoundTrip(request)
      }),
      CloseIdler: roundtrip.CloseIdlerFunc(func() {
        // do some decorator things here
        roundtrip.CloseIdleConnections(next)
      }),
    }
  }

The Chain type exposes CloseIdleConnections even if any chained constructors do not.

See: https://pkg.go.dev/net/http#Client.CloseIdleConnections

See: https://pkg.go.dev/net/http#RoundTripper
*/
package roundtrip
