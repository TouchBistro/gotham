package auth

import (
	"context"
	"time"

	"github.com/TouchBistro/gotham/ds"
)

// BasicPrincipal is a concrete implementation of Principal that mirrors the
// field layout of the legacy http.Principal struct. It serves as the default
// decode target for CachePrincipalLoader and the seed value for
// BasicPrincipalLoader.
type BasicPrincipal struct {
	// Id is the identifier for the principal, normally the "sub" claim.
	Id string `json:"id"`

	RoleSet    ds.Set    `json:"roles"`
	Admin      bool      `json:"isAdmin"`
	SuperAdmin bool      `json:"isSuperAdmin"`
	ExpiryAt   time.Time `json:"expiry"`
	Grps       []string  `json:"groups"`
}

// Identifier implements Principal.
func (p *BasicPrincipal) Identifier(ctx context.Context) string {
	return p.Id
}

// IsAdmin implements Principal.
func (p *BasicPrincipal) IsAdmin(ctx context.Context) bool {
	return p.Admin
}

// IsSuperAdmin implements Principal.
func (p *BasicPrincipal) IsSuperAdmin(ctx context.Context) bool {
	return p.SuperAdmin
}

// Roles implements Principal. Returns a sorted slice of role names.
func (p *BasicPrincipal) Roles(ctx context.Context) []string {
	return p.RoleSet.ToStringSlice()
}

// Groups implements Principal.
func (p *BasicPrincipal) Groups(ctx context.Context) []string {
	return p.Grps
}

// Expiry implements Principal.
func (p *BasicPrincipal) Expiry(ctx context.Context) time.Time {
	return p.ExpiryAt
}
