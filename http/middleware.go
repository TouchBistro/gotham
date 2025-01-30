package http

import "net/http"

// Middleware defines an interface to represent a net/http adapter, or middleware.
//
// The middleware implementationis required to Wrap(http.Handler) the supplied
// http.Handler with its own logic & return the new handelr
type Middleware interface {
	Wrap(next http.Handler) http.Handler
}

// MiddlewareFunc is a conveninece function implmenetation of Middleware
type MiddlewareFunc func(http.Handler) http.Handler

func (f MiddlewareFunc) Wrap(next http.Handler) http.Handler {
	return f(next)
}

// Chain returns a chain of http Handlers, where the middlewares are
// run in order of definition followed by the handler
func Chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i].Wrap(h)
	}
	return h
}
