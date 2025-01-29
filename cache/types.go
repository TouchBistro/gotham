package cache

import (
	"context"
	"fmt"
	"time"
)

const (
	NoExpiry time.Duration = 0
)

// MemoryCache interface definition for a cache abstraction
type MemoryCache interface {
	// store a key (string) value (any) in the cache without a set TTL
	Put(context.Context, string, any) error

	// store a key (string) value (any) in the cache with a TTL duration
	PutWithTtl(context.Context, string, any, time.Duration) error

	// fetch the value for the supplied key, return the value or error
	Fetch(context.Context, string, any) error

	// fetch the value and the set TTL for the supplied key or an error
	FetchWithTtl(context.Context, string, any) (*time.Duration, error)

	// delete the data for the supplied key
	Delete(context.Context, string) (int64, error)
}

// CacheMissError represents a cache miss; a defined err type so
// clients can distinguish between a cache-miss or other errors
type CacheMissError struct {
	key   string
	cause error
}

func (e *CacheMissError) Error() string {
	return fmt.Sprintf("cache-miss: no cached value found for key %v", e.key)
}
