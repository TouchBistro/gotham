package auth

import (
	"context"
	"testing"
	"time"
)

// Compile-time check: the factory returns something assignable to PrincipalLoader.
var _ PrincipalLoader = BasicPrincipalLoader(BasicPrincipal{})

func TestBasicPrincipalLoader_OverridesSub(t *testing.T) {
	ctx := context.Background()
	exp := time.Now().Add(time.Hour)

	seed := BasicPrincipal{
		Id:         "seed-id",
		Admin:      true,
		SuperAdmin: true,
		ExpiryAt:   exp,
	}

	loader := BasicPrincipalLoader(seed)
	out, err := loader.FetchPrincipal(ctx, FetchPrincipalInput{Id: "override-sub", PolicyConfig: Config{}})
	if err != nil {
		t.Fatalf("FetchPrincipal: %v", err)
	}
	got := out.Principal

	if id := got.Identifier(ctx); id != "override-sub" {
		t.Errorf("Identifier: got %q, want %q", id, "override-sub")
	}
	if !got.IsAdmin(ctx) {
		t.Error("IsAdmin: got false, want true")
	}
	if !got.IsSuperAdmin(ctx) {
		t.Error("IsSuperAdmin: got false, want true")
	}
	if e := got.Expiry(ctx); !e.Equal(exp) {
		t.Errorf("Expiry: got %v, want %v", e, exp)
	}
}

func TestBasicPrincipalLoader_DoesNotMutateSeed(t *testing.T) {
	ctx := context.Background()
	seed := BasicPrincipal{Id: "original"}

	loader := BasicPrincipalLoader(seed)
	_, _ = loader.FetchPrincipal(ctx, FetchPrincipalInput{Id: "sub-1"})
	_, _ = loader.FetchPrincipal(ctx, FetchPrincipalInput{Id: "sub-2"})

	if seed.Id != "original" {
		t.Errorf("seed was mutated: Id=%q, want %q", seed.Id, "original")
	}
}
