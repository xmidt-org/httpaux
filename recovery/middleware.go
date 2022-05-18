package recovery

import (
	"fmt"
	"io"
	"net/http"
	"runtime/debug"

	"github.com/xmidt-org/httpaux"
)

// OnRecover is a callback that receives information about a recovery object.
// Both the argument passed to panic and the debug stack trace are passed to this closure.
type OnRecover func(r interface{}, stack []byte)

// RecoverBody is a custom closure for writing recovery information.  By default,
// DefaultRecoverBody is used.
type RecoverBody func(w io.Writer, r interface{}, stack []byte)

// DefaultRecoverBody is the default strategy for writing the recovery argument.
// This function writes a string representation of r, followed by the stack trace.
func DefaultRecoverBody(w io.Writer, r interface{}, stack []byte) {
	_, err := fmt.Fprintf(w, "%s\n", r)
	if err == nil {
		w.Write(stack)
	}
}

type decorator struct {
	next http.Handler

	header     httpaux.Header
	body       RecoverBody
	statusCode int
	onRecover  []OnRecover
}

func (d *decorator) writeHeader(response http.ResponseWriter, r interface{}) {
	d.header.AddTo(response.Header())

	type headerer interface {
		Headers() http.Header
	}

	if h, ok := r.(headerer); ok {
		for name, values := range h.Headers() {
			for _, value := range values {
				response.Header().Add(name, value)
			}
		}
	}
}

func (d *decorator) writeStatusCode(response http.ResponseWriter, r interface{}) {
	type statusCoder interface {
		StatusCode() int
	}

	sc, ok := r.(statusCoder)
	switch {
	case ok:
		response.WriteHeader(sc.StatusCode())

	case d.statusCode >= 100:
		response.WriteHeader(d.statusCode)

	default:
		response.WriteHeader(http.StatusInternalServerError)
	}
}

func (d *decorator) writeBody(response http.ResponseWriter, r interface{}, stack []byte) {
	body := d.body
	if body == nil {
		body = DefaultRecoverBody
	}

	body(response, r, stack)
}

func (d *decorator) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			d.writeHeader(response, r)
			d.writeStatusCode(response, r)
			d.writeBody(response, r, stack)

			for _, f := range d.onRecover {
				f(r, stack)
			}
		}
	}()

	d.next.ServeHTTP(response, request)
}

// Option is a configurable option for a recovery decorator.
type Option interface {
	apply(*decorator)
}

type optionFunc func(*decorator)

func (of optionFunc) apply(d *decorator) { of(d) }

// WithOnRecover adds zero or more OnRecover callbacks to the middleware.
func WithOnRecover(f ...OnRecover) Option {
	return optionFunc(func(d *decorator) {
		d.onRecover = append(d.onRecover, f...)
	})
}

// WithRecoverBody adds a custom RecoverBody strategy for writing
// out recover objects.
func WithRecoverBody(rb RecoverBody) Option {
	return optionFunc(func(d *decorator) {
		d.body = rb
	})
}

// WithStatusCode sets a custom status code to use when a panic occurs.
func WithStatusCode(sc int) Option {
	return optionFunc(func(d *decorator) {
		d.statusCode = sc
	})
}

// WithHeader adds headers to write when a panic occurs.  This option
// is cumulative: headers from multiple calls will be merged together.
func WithHeader(h httpaux.Header) Option {
	return optionFunc(func(d *decorator) {
		d.header = d.header.Extend(h)
	})
}

// Middleware creates a http.Handler decorator that recovers any panics
// from downstream handlers.
//
// Decorated handlers created by the returned middleware will catch all panics
// and attempt to write a useful HTTP message.  By default, http.StatusInternalServerError
// will be set as the status, and the recovery object passed to panic will be output
// along with a debug stack trace.  The options can be used to customize this behavior.
//
// Note that if any handlers wrote information to the HTTP response before the panic,
// the decorator may not be able to write panic information.  For example, if WriteHeader
// was called prior to panicking, then the decorator cannot set a status code.
//
// In addition to HTTP output, one or more OnRecover strategies can be added via options
// that will be called with the recovery object and debug stack.  This can be used to hook
// in logging, metrics, etc.
func Middleware(options ...Option) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		d := &decorator{
			next: next,
		}

		for _, o := range options {
			o.apply(d)
		}

		return d
	}
}
