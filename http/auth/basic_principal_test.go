package auth

import (
	"context"
	"testing"
	"time"

	"github.com/TouchBistro/gotham/ds"
)

// Compile-time check: *BasicPrincipal satisfies Principal.
var _ Principal = (*BasicPrincipal)(nil)

func TestBasicPrincipal_InterfaceMethods(t *testing.T) {
	ctx := context.Background()
	exp := time.Now().Add(time.Hour)

	bp := &BasicPrincipal{
		Id:         "user-123",
		RoleSet:    ds.From("reader", "admin"),
		Admin:      true,
		SuperAdmin: false,
		ExpiryAt:   exp,
	}

	if got, want := bp.Identifier(ctx), "user-123"; got != want {
		t.Errorf("Identifier: got %q, want %q", got, want)
	}
	if !bp.IsAdmin(ctx) {
		t.Error("IsAdmin: got false, want true")
	}
	if bp.IsSuperAdmin(ctx) {
		t.Error("IsSuperAdmin: got true, want false")
	}
	if got, want := bp.Expiry(ctx), exp; !got.Equal(want) {
		t.Errorf("Expiry: got %v, want %v", got, want)
	}

	roles := bp.Roles(ctx)
	if len(roles) != 2 {
		t.Fatalf("Roles: got %d entries, want 2", len(roles))
	}
	// RoleSetFrom -> ToStringSlice returns sorted.
	if roles[0] != "admin" || roles[1] != "reader" {
		t.Errorf("Roles: got %v, want [admin reader]", roles)
	}
}
