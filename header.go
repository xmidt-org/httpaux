package httpaux

import "net/http"

// emptyHeader is the canonical, immutable empty Header
var emptyHeader = Header{}

// Header is a more efficient version of http.Header for situations where
// a number of HTTP headers are stored in memory and reused.  Rather than
// a map, a simple list of headers is maintained in canonicalized form.  This
// is much faster to iterate over than a map, which becomes important when
// the same Header is used to add headers to many requests.
//
// A Header instance is immutable once created.
type Header struct {
	names  []string
	values [][]string
}

// NewHeader creates an immutable, preprocessed Header given an
// http.Header
func NewHeader(v http.Header) Header {
	if len(v) > 0 {
		h := Header{
			names:  make([]string, 0, len(v)),
			values: make([][]string, 0, len(v)),
		}

		for name, values := range v {
			h.names = append(h.names, http.CanonicalHeaderKey(name))
			h.values = append(h.values, values)
		}

		return h
	}

	return emptyHeader
}

// NewHeaderFromMap allows a Header to be built directly from a map[string]string
// rather than an http.Header.
func NewHeaderFromMap(v map[string]string) Header {
	if len(v) > 0 {
		h := Header{
			names:  make([]string, 0, len(v)),
			values: make([][]string, 0, len(v)),
		}

		for name, value := range v {
			h.names = append(h.names, http.CanonicalHeaderKey(name))
			h.values = append(h.values, []string{value})
		}

		return h
	}

	return emptyHeader
}

// NewHeaders takes a variadic list of values and interprets them as alternating
// name/value pairs, with each pair specifying an HTTP header.  Duplicate header names
// are supported, which results in multivalued headers.  If v contains an odd number
// of strings, the last string is interpreted as a header with a blank value.
func NewHeaders(v ...string) Header {
	if len(v) > 0 {
		h := make(http.Header)

		var i, j int
		for i, j = 0, 1; j < len(v); i, j = i+2, j+2 {
			h.Add(v[i], v[j])
		}

		if i < len(v) {
			h.Add(v[i], "")
		}

		return NewHeader(h)
	}

	return emptyHeader
}

// SetTo overwrites headers in the destination with the ones defined by
// this Header.
func (h Header) SetTo(dst http.Header) {
	for i, name := range h.names {
		// the names are already canonicalized
		dst[name] = h.values[i]
	}
}

// Then is a server middleware that adds all the headers to the http.ResponseWriter
// prior to invoking the next handler.  As an optimization, if this Header is empty
// no decoration is done.
func (h Header) Then(next http.Handler) http.Handler {
	if len(h.names) > 0 {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			// have to set headers first, as the next handler will likely invoke WriteHeader or Write
			h.SetTo(response.Header())
			next.ServeHTTP(response, request)
		})
	}

	return next
}

// headerRoundTripper is a http.RoundTripper decorator returned by Header.RoundTrip
type headerRoundTripper struct {
	h    Header
	next http.RoundTripper
}

func (hrt headerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	hrt.h.SetTo(request.Header)
	return hrt.next.RoundTrip(request)
}

// RoundTrip is a client middleware that adds all these headers to the http.Request
// prior to invoking the next http.RoundTripper.  As an optimization, if this Header
// is empty no decoration is done.  Next is returned as is in that case.
//
// If next is nil and this Header is non-empty, then http.DefaultTransport is decorated.
func (h Header) RoundTrip(next http.RoundTripper) http.RoundTripper {
	if len(h.names) > 0 {
		if next == nil {
			next = http.DefaultTransport
		}

		return headerRoundTripper{
			h:    h,
			next: next,
		}
	}

	return next
}
