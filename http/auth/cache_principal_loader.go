package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/TouchBistro/gotham/cache"
	log "github.com/sirupsen/logrus"
)

// CachePrincipalLoader is a PrincipalLoader backed by a cache.MemoryCache. The
// type parameter T is the concrete Principal implementation the cache will
// decode entries into. Because the backing cache (e.g. cache.RamCache) may rely
// on reflection-based type matching for in-memory stores, T should be the same
// concrete type used when writing entries via Persist — typically a pointer
// type such as *BasicPrincipal.
type CachePrincipalLoader[T Principal] struct {
	// KeyPrefix is prepended to the subject when constructing the cache key.
	KeyPrefix string

	// Cache is the backing memory cache.
	Cache cache.MemoryCache
}

// FetchPrincipal implements PrincipalLoader. It allocates a zero value of T,
// asks the cache to populate it, and returns it as a Principal. The type
// constraint guarantees T itself satisfies Principal, so no assertion is
// needed on the return.
func (l CachePrincipalLoader[T]) FetchPrincipal(ctx context.Context, in FetchPrincipalInput) (FetchPrincipalOutput, error) {
	key := l.buildCacheKey(in.Id)

	var t T
	ttl, err := l.Cache.FetchWithTtl(ctx, key, &t)
	if err != nil {
		log.Debugf("cache miss for key=%v (sub) when fetching cached principal", key)
		return FetchPrincipalOutput{}, err
	}

	log.Debugf("cache hit for key=%v (sub) ttl=%v", key, time.Now().Add(*ttl))
	return FetchPrincipalOutput{Principal: t}, nil
}

// Persist writes a Principal into the cache, keyed on its Identifier and with
// a TTL derived from its Expiry.
func (l CachePrincipalLoader[T]) Persist(ctx context.Context, pr Principal) error {
	key := l.buildCacheKey(pr.Identifier(ctx))
	ttl := time.Until(pr.Expiry(ctx))
	log.Debugf("caching principal key=%v (sub), ttl=%v", key, ttl)

	return l.Cache.PutWithTtl(ctx, key, pr, ttl)
}

func (l CachePrincipalLoader[T]) buildCacheKey(sub string) string {
	if l.KeyPrefix != "" {
		return fmt.Sprintf("%v::%v", l.KeyPrefix, sub)
	}
	return sub
}
