// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package busy

import (
	"net/http"
)

// Server defines a server middleware that enforces request limiting
type Server struct {
	// Limiter is the concurrent request limiting strategy.  If this field is unset,
	// then no limiting is done.
	Limiter Limiter

	// Busy is the optional http.Handler to invoke when the maximum number
	// concurrent requests has been exceeded.  ConstantHandler is a useful choice
	// for this field, as it allows one to tailor not only the status code but also
	// the headers and body.
	//
	// If this field is nil, this middleware simply returns http.StatusServiceUnavailable.
	Busy http.Handler
}

// Then is a server middleware that enforces this busy configuration.  If Limiter is nil,
// no decoration is done and next is returned as is.  If OnBusy is nil, then the returned
// handler will simply set http.StatusServiceUnavailable when requests fail the limit check.
func (s Server) Then(next http.Handler) http.Handler {
	if s.Limiter == nil {
		return next
	}

	return &busyDecorator{
		Server: s,
		next:   next,
	}
}

// busyDecorator is a decorator around a next handler that enforces a request limit
type busyDecorator struct {
	Server
	next http.Handler
}

func (bd *busyDecorator) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if done, ok := bd.Limiter.Check(request); ok {
		defer done()
		bd.next.ServeHTTP(response, request)
	} else if bd.Busy != nil {
		bd.Busy.ServeHTTP(response, request)
	} else {
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}
