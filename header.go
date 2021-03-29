package httpaux

import (
	"net/http"
	"sort"
)

// emptyHeader is the canonical, immutable empty Header
var emptyHeader = Header{}

// header is the tuple of a canonicalized name together with its values.
type header struct {
	name   string
	values []string
}

// headers is a sortable slice of header objects.  this type is sortable
// so that when merging, it can be searched for existing header names.
type headers []header

func (h headers) Len() int           { return len(h) }
func (h headers) Less(i, j int) bool { return h[i].name < h[j].name }
func (h headers) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// clone makes a distinct copy, ensuring that the clone has the given
// minimim capacity
func (h headers) clone(minCap int) headers {
	newCap := len(h)
	if newCap < minCap {
		newCap = minCap
	}

	clone := make(headers, len(h), newCap)
	copy(clone, h)
	return clone
}

// sort resorts this header ascending according to name
func (h headers) sort() {
	sort.Sort(h)
}

// add handles adding a set of values for a name.  if the name exists,
// the values are appended.  if the name was not found, it is inserted in
// such a way as to preserve the order property.  the values slice is
// copied into the header entry.
//
// add canonicalizes name prior to insertion.
func (h *headers) add(name string, values []string) {
	if len(values) == 0 {
		// optimization: don't bother if there are no values
		return
	}

	name = http.CanonicalHeaderKey(name)
	n := len(*h)
	p := sort.Search(n, func(i int) bool {
		return (*h)[i].name >= name
	})

	if p == n || (*h)[p].name != name {
		if cap(*h) < n+1 {
			// we need to allocate
			grow := make(headers, n+1)
			copy(grow, (*h)[0:p])

			if p < n {
				copy(grow[p+1:], (*h)[p+1:])
			}

			*h = grow
		} else {
			// can grow in place
			*h = (*h)[0 : n+1]
			copy((*h)[p+1:], (*h)[p:])
		}

		// since this is a new entry, need to null out the old values
		(*h)[p].values = nil
	}

	// now, p is always the right spot to insert the new name/values tuple
	(*h)[p].name = name
	merged := make([]string, 0, len((*h)[p].values)+len(values))
	merged = append(merged, (*h)[p].values...)
	merged = append(merged, values...)
	(*h)[p].values = merged
}

// Header is a more efficient version of http.Header for situations where
// a number of HTTP headers are stored in memory and reused.  Rather than
// a map, a simple list of headers is maintained in canonicalized form.  This
// is much faster to iterate over than a map, which becomes important when
// the same Header is used to add headers to many requests.
//
// A Header instance is immutable once created.  The zero value is an
// usable, empty Header which is immutable.
type Header struct {
	h headers
}

// EmptyHeader returns a canonical singleton Header that is empty.
// Useful as a "nil" value or as a default.
func EmptyHeader() Header {
	return emptyHeader
}

// Len returns the number of distinct header names in this Header.
func (h Header) Len() int {
	return h.h.Len()
}

// Append returns a new Header instance with the given http.Header
// values appended to these header values.  If more is empty, the original
// Header is returned.
//
// This method does not modify the original Header.
func (h Header) Append(more http.Header) Header {
	if len(more) == 0 {
		return h
	}

	n := Header{
		h: h.h.clone(h.Len() + len(more)),
	}

	for name, values := range more {
		n.h.add(name, values)
	}

	return n
}

// AppendMap returns a new Header instance with the given map of headers
// added.  This function is useful in the simple cases where only single-valued
// headers are supported by application code.  If more is empty, the original
// Header is returned.
//
// This method does not modify the original Header.
func (h Header) AppendMap(more map[string]string) Header {
	if len(more) == 0 {
		return h
	}

	n := Header{
		h: h.h.clone(h.Len() + len(more)),
	}

	n.h = append(n.h, h.h...)
	for name, value := range more {
		n.h.add(name, []string{value})
	}

	return n
}

// AppendHeaders return a new Header with the variadic slice of name/value
// pairs appended.  If more has an odd length, the last value is assumed to
// be a name with a blank value.  If more is empty, the original Header
// is returned.
//
// This method does not modify the original Header.
func (h Header) AppendHeaders(more ...string) Header {
	if len(more) == 0 {
		return h
	}

	n := Header{
		h: h.h.clone(h.Len() + 1 + len(more)/2), // worst case
	}

	for i, j := 0, 1; i < len(more); i, j = i+2, j+2 {
		var value string
		if j < len(more) {
			value = more[j]
		}

		n.h.add(more[i], []string{value})
	}

	return n
}

// Extend returns the merger of this Header with the given Header.  If
// this Header is empty, more is returned.  If more is empty, this Header
// is returned.  Otherwise, a new Header that is the union of the two
// is returned.
//
// This method does not modify the original Header.
func (h Header) Extend(more Header) Header {
	if more.Len() == 0 {
		return h
	} else if h.Len() == 0 {
		return more
	}

	n := Header{
		h: h.h.clone(h.Len() + more.Len()),
	}

	n.h = append(n.h, more.h...)
	n.h.sort()
	return n
}

// NewHeader creates an immutable, preprocessed Header given an http.Header.
func NewHeader(v http.Header) Header {
	return emptyHeader.Append(v)
}

// NewHeaderFromMap allows a Header to be built directly from a map[string]string
// rather than an http.Header.
func NewHeaderFromMap(v map[string]string) Header {
	return emptyHeader.AppendMap(v)
}

// NewHeaders takes a variadic list of values and interprets them as alternating
// name/value pairs, with each pair specifying an HTTP header.  Duplicate header names
// are supported, which results in multivalued headers.  If v contains an odd number
// of strings, the last string is interpreted as a header with a blank value.
func NewHeaders(v ...string) Header {
	return emptyHeader.AppendHeaders(v...)
}

// SetTo replaces each name/value defined in this Header in the supplied http.Header.
func (h Header) SetTo(dst http.Header) {
	for _, h := range h.h {
		dst[h.name] = h.values // TODO: safe copy?
	}
}

// AddTo adds each name/value defined in this Header to the supplied
// http.Header.
func (h Header) AddTo(dst http.Header) {
	for _, h := range h.h {
		dst[h.name] = append(dst[h.name], h.values...)
	}
}

// Then is a server middleware that adds all the headers to the http.ResponseWriter
// prior to invoking the next handler.  As an optimization, if this Header is empty
// no decoration is done.
//
// This method can be used with libraries like justinas/alice:
//
//   h := httpaux.NewHeader("Header", "Value")
//   c := alice.New(
//     h.Then,
//   )
func (h Header) Then(next http.Handler) http.Handler {
	if h.Len() > 0 {
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
//
// This method can be used as a roundtrip.Constructor or as part of a roundtrip.Chain:
//
//   h := httpaux.Header("Header", "Value")
//   c := roundtrip.NewChain(
//     h.RoundTrip,
//   )
func (h Header) RoundTrip(next http.RoundTripper) http.RoundTripper {
	if h.Len() > 0 {
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
