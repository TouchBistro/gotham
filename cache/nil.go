package cache

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// NilCache a MemoryCache implementation for no-op cache
type NilCache struct{}

// method implementations

func (r *NilCache) Put(ctx context.Context, key string, val any) error {
	return nil
}

func (r *NilCache) PutS(ctx context.Context, key string, val string) error {
	return nil
}

func (r *NilCache) PutWithTtl(ctx context.Context, key string, val any, expiry time.Duration) error {
	return nil
}

func (r *NilCache) PutWithTtlS(ctx context.Context, key string, val string, expiry time.Duration) error {
	return nil
}

func (r *NilCache) Fetch(ctx context.Context, key string, val any) error {
	return &CacheMissError{key, errors.Errorf("cache miss, nil cache in use")}
}

func (r *NilCache) FetchWithTtl(ctx context.Context, key string, val any) (*time.Duration, error) {
	return nil, &CacheMissError{key, errors.Errorf("cache miss, nil cache in use")}
}

func (r *NilCache) Delete(ctx context.Context, key string) (int64, error) {
	return 0, nil
}
