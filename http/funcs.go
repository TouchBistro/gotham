package http

import (
	"net/http"

	"github.com/pkg/errors"
)

// httpRequestHeaderValue returns the n-th value (0-based index) for the supplied
// request header name
func httpRequestHeaderValue(r *http.Request, name string, index int) (string, error) {
	vals := r.Header[name]
	if index > len(vals)-1 {
		return "", errors.Errorf("no value for header %v index %v exists", name, index)
	}
	return vals[index], nil
}
