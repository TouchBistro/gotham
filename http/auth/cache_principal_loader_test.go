package auth

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/TouchBistro/gotham/cache"
)

// Compile-time checks: both pointer-typed instantiations implement PrincipalLoader.
var (
	_ PrincipalLoader = CachePrincipalLoader[*BasicPrincipal]{}
)

// fakeCache is a minimal in-memory cache.MemoryCache used for testing
// CachePrincipalLoader. Unlike cache.RamCache, it keeps the stored value
// as-is and uses reflection to assign it back into the caller's pointer,
// mimicking RamCache's behavior without depending on its internals.
type fakeCache struct {
	entries map[string]any
	ttl     time.Duration
}

func newFakeCache() *fakeCache {
	return &fakeCache{entries: map[string]any{}, ttl: time.Hour}
}

func (c *fakeCache) Put(ctx context.Context, key string, val any) error {
	return c.PutWithTtl(ctx, key, val, 0)
}

func (c *fakeCache) PutWithTtl(ctx context.Context, key string, val any, _ time.Duration) error {
	c.entries[key] = val
	return nil
}

func (c *fakeCache) Fetch(ctx context.Context, key string, val any) error {
	stored, ok := c.entries[key]
	if !ok {
		return errors.New("cache miss")
	}
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Pointer {
		return errors.New("target must be a pointer")
	}
	ele := rv.Elem()
	sv := reflect.ValueOf(stored)
	if !sv.Type().AssignableTo(ele.Type()) {
		return errors.New("type mismatch")
	}
	ele.Set(sv)
	return nil
}

func (c *fakeCache) FetchWithTtl(ctx context.Context, key string, val any) (*time.Duration, error) {
	if err := c.Fetch(ctx, key, val); err != nil {
		return nil, err
	}
	d := c.ttl
	return &d, nil
}

func (c *fakeCache) Delete(ctx context.Context, key string) (int64, error) {
	delete(c.entries, key)
	return 0, nil
}

// Confirm the fake satisfies the interface.
var _ cache.MemoryCache = (*fakeCache)(nil)

func TestCachePrincipalLoader_RoundTrip(t *testing.T) {
	ctx := context.Background()
	exp := time.Now().Add(30 * time.Minute)

	pr := &BasicPrincipal{
		Id:       "sub-abc",
		Admin:    true,
		ExpiryAt: exp,
	}

	loader := CachePrincipalLoader[*BasicPrincipal]{
		KeyPrefix: "test",
		Cache:     newFakeCache(),
	}

	if err := loader.Persist(ctx, pr); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	out, err := loader.FetchPrincipal(ctx, FetchPrincipalInput{Id: "sub-abc", PolicyConfig: Config{}})
	if err != nil {
		t.Fatalf("FetchPrincipal: %v", err)
	}
	got := out.Principal

	if id := got.Identifier(ctx); id != "sub-abc" {
		t.Errorf("Identifier: got %q, want %q", id, "sub-abc")
	}
	if !got.IsAdmin(ctx) {
		t.Error("IsAdmin: got false, want true")
	}
	if e := got.Expiry(ctx); !e.Equal(exp) {
		t.Errorf("Expiry: got %v, want %v", e, exp)
	}
}

func TestCachePrincipalLoader_MissReturnsError(t *testing.T) {
	loader := CachePrincipalLoader[*BasicPrincipal]{
		Cache: newFakeCache(),
	}
	if _, err := loader.FetchPrincipal(context.Background(), FetchPrincipalInput{Id: "missing", PolicyConfig: Config{}}); err == nil {
		t.Error("FetchPrincipal: expected error on miss, got nil")
	}
}

func TestCachePrincipalLoader_KeyPrefixing(t *testing.T) {
	cases := []struct {
		prefix string
		sub    string
		want   string
	}{
		{prefix: "", sub: "u1", want: "u1"},
		{prefix: "p", sub: "u1", want: "p::u1"},
	}
	for _, c := range cases {
		l := CachePrincipalLoader[*BasicPrincipal]{KeyPrefix: c.prefix}
		if got := l.buildCacheKey(c.sub); got != c.want {
			t.Errorf("buildCacheKey(prefix=%q, sub=%q) = %q, want %q", c.prefix, c.sub, got, c.want)
		}
	}
}
