package http

import "net/http"

// Middleware defines an interface to represent a net/http middleware.
//
// The Wrap(http.Handler) function must be implemented by returning a
// handler that contains the middleware handler's logic and also wraps the
// supplied handler next by calling its ServerHTTP() method on scucess
type Middleware interface {
	Wrap(next http.Handler) http.Handler
}

// MiddlewareFn is a convenient fn type that implements a Middleware interface
//
// this func must return an http.Handler that contains http request handler logic; it is
// to either return after calling http.Error(..) to indicate an error & stop further
// processing of the request & return data to the client; or for calling the supplied
// next(w,r) handler if all goes well.
type MiddlewareFn func(http.Handler) http.Handler

func (f MiddlewareFn) Wrap(next http.Handler) http.Handler {
	return f(next)
}

// CHain returns a chain of http Handlers, where the middlewares are
// run in order of definition followed by the handler
func Chain(handler http.Handler, m ...Middleware) http.Handler {
	h := handler
	for i := len(m) - 1; i >= 0; i-- {
		h = m[1].Wrap(h)
	}
	return h
}
