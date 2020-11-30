package observe

import "net/http"

type observableDecorator struct {
	next http.Handler
}

func (od observableDecorator) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	od.next.ServeHTTP(
		New(response),
		request,
	)
}

// Then is a serverside middleware that ensures the next handler sees
// an observable Writer in it ServeHTTP method.  This method is idempotent.
// If next is already an observable handler, it is returned as is.  Additionally,
// if an http.ResponseWriter is already observable, possibly due to other infrastructure,
// the returned handler will simply pass the call through to next.
func Then(next http.Handler) http.Handler {
	if _, ok := next.(observableDecorator); ok {
		return next
	}

	return observableDecorator{
		next: next,
	}
}
