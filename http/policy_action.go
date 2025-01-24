package http

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type FieldType string

const (
	FieldTypeHeader FieldType = "header"
)

type PolicyAction struct {
	Type   FieldType `json:"type"` // The type to apply the action one, currently supported header
	Fn     string    `json:"fn"`
	Params []string  `json:"params"`
}

func (p PolicyAction) toGinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		p.apply(c.Request)
	}
}

func (p PolicyAction) toHttpHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p.apply(r)
	}
}

func (p PolicyAction) toHttpMiddlewareFunc() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p.apply(r)
			next.ServeHTTP(w, r)
		})
	}
}

// apply evaluates the action on the supplied http request
func (p PolicyAction) apply(r *http.Request) error {
	var typ any
	switch p.Type {
	case FieldTypeHeader:
		typ = r.Header
	default:
		log.Warnf("unsupported action %v supplied, ignored", p.Type)
	}

	// convert parms to []reflect.Value
	v := make([]reflect.Value, 0)
	for _, p := range p.Params {
		v = append(v, reflect.ValueOf(p))
	}

	valOfType := reflect.ValueOf(typ)
	met := valOfType.MethodByName(p.Fn)
	_ = met.Call(v) // call the method
	return nil
}
