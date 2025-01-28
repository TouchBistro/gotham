package cache

import (
	"context"
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

func (r RedisCache) internalSingletonKey() string {
	return fmt.Sprintf("%v:%v/%v", r.host, r.port, r.db)
}

// func (r *RedisCache) ser(o any) ([]byte, error) {
// 	var buf bytes.Buffer
// 	enc := gob.NewEncoder(&buf)
// 	err := enc.Encode(o)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return buf.Bytes(), nil
// }

// func (r *RedisCache) de(data []byte, obj any) error {
// 	buf := bytes.NewBuffer(data)
// 	dec := gob.NewDecoder(buf)
// 	return dec.Decode(obj)
// }

// method implementations

func (r *RedisCache) Put(ctx context.Context, key string, val any) error {
	return r.PutWithTtl(ctx, key, val, NoExpiry)
}

func (r *RedisCache) PutWithTtl(ctx context.Context, key string, val any, expiry time.Duration) error {
	var bytes []byte
	switch rval := val.(type) {
	case []byte:
		bytes = rval //val.([]byte)
	case string:
		bytes = []byte(rval)
	default:
		var err error
		g := GobSerde{}
		if bytes, err = g.ser(val); err != nil {
			return err
		}
		// if bytes, err = GobSerde{}.ser(val); err != nil {
		// 	return err
		// }
	}

	cmd := r.client.Set(ctx, key, bytes, expiry)
	if _, err := cmd.Result(); err != nil {
		return err
	}

	return nil
}

func (r *RedisCache) Fetch(ctx context.Context, key string, val any) error {
	if bytes, err := r.client.Get(ctx, key).Bytes(); err != nil {
		return err
	} else {
		switch rval := val.(type) {
		// case []byte: // todo remove
		// 	val = bytes
		// 	return nil
		case *[]byte:
			*rval = bytes
			return nil
		// case string: // todo remove
		// 	val = string(bytes)
		// 	return nil
		case *string:
			str := string(bytes)
			val = &str
			return nil
		default:
			return GobSerde{}.de(bytes, val) //TODO to be improved into a more generic SerDe impl
		}
	}
}

func (r *RedisCache) FetchWithTtl(ctx context.Context, key string, val any) (*time.Duration, error) {

	if err := r.Fetch(ctx, key, val); err != nil {
		return nil, err
	}

	var expiry *time.Duration
	cmd2 := r.client.TTL(ctx, key)
	if ttl, err := cmd2.Result(); err != nil {
		return nil, err
	} else {
		expiry = &ttl
	}

	return expiry, nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) (int64, error) {
	cmd := r.client.Del(ctx, key)
	if n, err := cmd.Result(); err != nil {
		return 0, err
	} else {
		return n, nil
	}
}
