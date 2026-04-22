package auth

import (
	"net/http"
	"reflect"

	log "github.com/sirupsen/logrus"
)

type FieldType string

const (
	FieldTypeHeader FieldType = "header"
)

// Action describes a pre/post action applied to a request during auth
// processing (e.g. mutating a request header).
type Action struct {
	Type   FieldType `json:"type"`
	Fn     string    `json:"fn"`
	Params []string  `json:"params"`
}

// Apply evaluates the action on the supplied http request.
func (a Action) Apply(r *http.Request) error {
	var typ any
	switch a.Type {
	case FieldTypeHeader:
		typ = r.Header
	default:
		log.Warnf("unsupported action %v supplied, ignored", a.Type)
	}

	v := make([]reflect.Value, 0)
	for _, p := range a.Params {
		v = append(v, reflect.ValueOf(p))
	}

	valOfType := reflect.ValueOf(typ)
	met := valOfType.MethodByName(a.Fn)
	_ = met.Call(v)
	return nil
}
