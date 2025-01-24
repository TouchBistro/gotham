package http

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
)

const (
	ContextMapKey       string = "context-map" // key used by net/http handlers to store a context map context wrapper value map
	ContextKeyAlias     string = "alias"
	ContextKeyPrincipal string = "principal"
)

// value_map checks if the context has a Value of the required map type
func value_map(ctx context.Context) (map[string]any, error) {
	_cmap := ctx.Value(ContextMapKey)
	if _cmap == nil {
		return nil, errors.Errorf("the context does not have a map value")
	}

	cmap, ok := _cmap.(map[string]any)
	if !ok {
		return cmap, errors.Errorf("the context does not have a value map value for key %v", ContextMapKey)
	}

	return cmap, nil
}

// setValue sets the key/value in the internal map, returns a non-nil error if the
// context-map is not set for this context.Value() or is not the correct type
func setValue(ctx context.Context, key string, val any) error {
	// if vmap, err := value_map(ctx); err != nil {
	if vmap, err := value_map(ctx); err != nil {
		return err
	} else {
		vmap[key] = val
	}
	return nil
}

// getValue returns the value stored in the internal map for the supplied key, else err
func getValue(ctx context.Context, key string) (any, error) {
	var val any
	if vmap, err := value_map(ctx); err != nil {
		return nil, err
	} else {
		var ok bool
		if val, ok = vmap[key]; !ok {
			return nil, errors.Errorf("no value exists for key %v", key)
		}
	}
	return val, nil
}

// upgradeRequestContext upgrades the http.Request with a new context that contains
// the value map
func upgradeRequestContext(r *http.Request) *http.Request {
	if _, err := value_map(r.Context()); err != nil {
		return r.WithContext(context.WithValue(r.Context(), ContextMapKey, make(map[string]any)))
	}
	return r
}
