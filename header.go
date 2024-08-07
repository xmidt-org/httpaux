// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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

func (h headers) Len() int { return len(h) }

// clone makes a distinct copy, ensuring that the clone has the given
// minimum capacity.
//
// IMPORTANT: this method MUST be called before adding a series of
// name/value pairs in order to grow the headers large enough to handle them.
func (h headers) clone(minCap int) headers {
	newCap := len(h)
	if newCap < minCap {
		newCap = minCap
	}

	clone := make(headers, len(h), newCap)
	copy(clone, h)
	return clone
}

// add handles adding a set of values for a name.  if the name exists,
// the values are appended.  if the name was not found, it is inserted in
// such a way as to preserve the order property.  the values slice is
// copied into the header entry.
//
// add canonicalizes name prior to insertion.
//
// this method assumes clone has been called, so that the capacity
// is large enough to hold the name/values entry.  this method panics
// if that is not the case.
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
		// grow in place, which assumes clone had previously been called
		// this will panic if not
		*h = (*h)[0 : n+1]
		copy((*h)[p+1:], (*h)[p:])

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

	for name, value := range more {
		n.h.add(name, []string{value})
	}

	return n
}

// AppendHeaders return a new Header with the variadic slice of name/value
// pairs appended.  If more has an odd length, the last value is assumed to
// be a name with a blank value.  If more is empty, the original Header
// is returned.  Duplicate names are allowed, which result in multi-valued headers.
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

	for _, he := range more.h {
		n.h.add(he.name, he.values)
	}

	return n
}

// NewHeader creates an immutable, preprocessed Header given an http.Header.
//
// If v is empty, the canonical EmptyHeader is returned.
func NewHeader(v http.Header) Header {
	return emptyHeader.Append(v)
}

// NewHeaderFromMap creates an immutable, preprocessed Header given a simple
// map of string-to-string.
//
// If v is empty, the canonical EmptyHeader is returned.
func NewHeaderFromMap(v map[string]string) Header {
	return emptyHeader.AppendMap(v)
}

// NewHeaders creates an immutable, preprocessed Header given a variadic list
// of names and values in {name1, value1, name2, value2, ...} order.  Duplicate
// names are allowed, which will result in multi-valued headers.  If v contains
// an odd number of values, the last value is interpreted as a header name with
// a single blank value.
//
// If v is empty, the canonical EmptyHeader is returned.
func NewHeaders(v ...string) Header {
	return emptyHeader.AppendHeaders(v...)
}

// SetTo replaces each name/value defined in this Header in the supplied http.Header.
func (h Header) SetTo(dst http.Header) {
	for _, h := range h.h {
		dst[h.name] = append([]string{}, h.values...) // preserve immutability
	}
}

// AddTo adds each name/value defined in this Header to the supplied
// http.Header.
func (h Header) AddTo(dst http.Header) {
	for _, h := range h.h {
		dst[h.name] = append(dst[h.name], h.values...)
	}
}
