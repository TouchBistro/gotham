package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

// RedisCache is a MemoryCache implementation for Redis database
type RedisCache struct {
	client   *redisv9.Client
	host     string
	port     int
	password string
	db       int
}

// connect to redis, or return error
func (r *RedisCache) connect() error {
	r.client = redisv9.NewClient(&redisv9.Options{
		Addr:     fmt.Sprintf("%v:%v", r.host, r.port), // host:port
		Password: r.password,                           // no password set
		DB:       r.db,                                 // use default DB, 0
	})

	if _, err := r.client.Ping(context.Background()).Result(); err != nil {
		return err
	}
	return nil
}

// method implementations

func (r *RedisCache) Put(ctx context.Context, key string, val any) error {
	return r.PutWithTtl(ctx, key, val, NoExpiry)
}

func (r *RedisCache) PutWithTtl(ctx context.Context, key string, val any, expiry time.Duration) error {

	bytes, err := json.Marshal(val)
	if err != nil {
		return err
	}

	cmd := r.client.Set(ctx, key, bytes, expiry)
	if _, err := cmd.Result(); err != nil {
		return err
	}
	return nil
}

func (r *RedisCache) Fetch(ctx context.Context, key string) (any, error) {
	cmd := r.client.Get(ctx, key)
	v, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	var val any
	if err := json.Unmarshal([]byte(v), &val); err != nil {
		return nil, err
	}
	return val, nil
}

func (r *RedisCache) FetchWithTtl(ctx context.Context, key string) (any, *time.Duration, error) {

	val, err := r.Fetch(ctx, key)
	if err != nil {
		return nil, nil, err
	}

	var expiry *time.Duration
	cmd2 := r.client.TTL(ctx, key)
	if ttl, err := cmd2.Result(); err != nil {
		return nil, nil, err
	} else {
		expiry = &ttl
	}

	return val, expiry, nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) (int64, error) {
	cmd := r.client.Del(ctx, key)
	if n, err := cmd.Result(); err != nil {
		return 0, err
	} else {
		return n, nil
	}
}
