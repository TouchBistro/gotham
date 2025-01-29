package cache

import (
	"context"
	"reflect"
	"time"

	"github.com/pkg/errors"
)

// RamCache a MemoryCache implementation for internal memory
type RamCache struct {
	rmap map[string]any
}

func (r *RamCache) initialize() {
	r.rmap = make(map[string]any)
}

// method implementations

func (r *RamCache) Put(ctx context.Context, key string, val any) error {
	return r.PutWithTtl(ctx, key, val, NoExpiry)
}

func (r *RamCache) PutWithTtl(ctx context.Context, key string, val any, expiry time.Duration) error {
	r.rmap[key] = val
	return nil
}

func (r *RamCache) Fetch(ctx context.Context, key string, val any) error {
	// the supplied val must be a pointer
	value_of_val := reflect.ValueOf(val)
	if value_of_val.Kind() != reflect.Pointer {
		return errors.New("attemp to Fetch into a non-pointer")
	}

	// let's fetch the value first...
	if _val, ok := r.rmap[key]; ok {
		// TODO the value is found, we return it via val (interface{}) by ref...
		ele := value_of_val.Elem() // value in val (any)
		if !reflect.TypeOf(_val).AssignableTo(ele.Type()) {
			return errors.Errorf("value of type %v cannot be assigned to type %v", reflect.TypeOf(_val), ele.Type())
		}
		ele.Set(reflect.ValueOf(_val))
		return nil
	}
	return &CacheMissError{key, errors.Errorf("key not found in ram cache")}
}

func (r *RamCache) FetchWithTtl(ctx context.Context, key string, val any) (*time.Duration, error) {
	if err := r.Fetch(ctx, key, val); err != nil {
		return nil, err
	}

	exp := time.Hour * 24 * 365 * 290
	return &exp, nil
}

func (r *RamCache) Delete(ctx context.Context, key string) (int64, error) {
	delete(r.rmap, key)
	return 0, nil
}
