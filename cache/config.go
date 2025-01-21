package cache

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type MemoryCacheKind string

const (
	InternalMemory MemoryCacheKind = "memory"
	Redis          MemoryCacheKind = "redis"
)

type Config struct {
	Kind        MemoryCacheKind `json:"kind"`
	RedisConfig *RedisConfig    `json:"redis-config"`
}

type RedisConfig struct {
	Host string `json:"host"`
	Port *int   `json:"port"`
	Db   int    `json:"db"`
}

var redisCacheImplMap map[string]MemoryCache

// Initialize a new instance of MemoryCache from app settings
// see InitializeWithConfig for impelementation details.
func Initialize() (MemoryCache, error) {
	return InitializeWithConfig(nil)
}

// InitializeWithConfig a new instace of a MemoryCache with the supplied config
// If nil config is supplied, the configuration is read from app settings
// using viper, or defaults are used.
//
// cache:
//
//	kind: redis  # redis|memory|memcached?
//	redis_config:
//	  host: localhost
//	  port: 6379
//	  db: 0
//
// A memory cache impl is initialized & returns, else a non-nil error
func InitializeWithConfig(cfg *Config) (MemoryCache, error) {

	config := cfg
	if config == nil {
		c := loadCacheConfigFromAppSettings()
		config = &c
	}

	switch config.Kind {

	// redis

	case Redis:

		// ultimate defaults
		host := "localhost"
		port := 6379
		db := 0

		// now check config
		if config.RedisConfig != nil {
			r := config.RedisConfig
			host = r.Host
			if r.Port != nil {
				port = *r.Port
			}
			db = r.Db
		}

		c := &RedisCache{
			host: host,
			port: port,
			db:   db,
		}

		// check singleton map, if an instance exists, then return it
		if c, ok := redisCacheImplMap[c.internalSingletonKey()]; ok {
			return c, nil
		}

		log.Debugf("initializing redis cache with host: %v, port: %v, db: %v", host, port, db)
		if err := c.connect(); err != nil {
			return nil, err
		}

		// initialize map
		if redisCacheImplMap == nil {
			redisCacheImplMap = make(map[string]MemoryCache)
		}

		// store singleton in map
		redisCacheImplMap[c.internalSingletonKey()] = c
		return c, nil

	// internal memory (RAM)
	case InternalMemory:
		return nil, fmt.Errorf("cache type %v not supported", config.Kind)

	// default
	default:
		return nil, fmt.Errorf("cache type %v not supported", config.Kind)

	}
}

// loadCacheConfigFromAppSettings loads cache configurationf from app settings
// user viper. The settings must be supplied using the json schema shown above
// which translates to the following json key path
//
// kind: cache.kind (string)
// redis host: cache.redis_config.host (string)
// redis port: cache.redis_config.port (int)
// redis db: cache.redis_config.db (int)
func loadCacheConfigFromAppSettings() Config {

	cfg := Config{
		Kind: Redis,
	}

	if viper.IsSet("cache.kind") {
		cfg.Kind = MemoryCacheKind(viper.GetString("cache.kind"))
		if cfg.Kind == Redis {
			host := "localhost"
			if viper.IsSet("cache.redis_config.host") {
				host = viper.GetString("cache.redis_config.host")
			}
			port := 6379
			if viper.IsSet("cache.redis_config.port") {
				port = viper.GetInt("cache.redis_config.port")
			}
			db := 0
			if viper.IsSet("cache.redis_config.db") {
				db = viper.GetInt("cache.redis_config.db")
			}

			cfg.RedisConfig = &RedisConfig{
				Host: host,
				Port: &port,
				Db:   db,
			}
		}
	}
	return cfg
}
