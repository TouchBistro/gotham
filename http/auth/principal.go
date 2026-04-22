// Package auth defines the Principal abstraction used by HTTP authentication
// middleware. A Principal represents the authenticated actor making a request
// and exposes the minimum behavior needed for authorization decisions.
package auth

import (
	"context"
	"time"
)

// Principal is the authenticated actor on a request. Implementations provide
// identity, role membership, admin flags, and an expiry time.
type Principal interface {
	// Identifier returns the stable identifier for this principal, typically
	// the value of the "sub" claim in a JWT.
	Identifier(ctx context.Context) string

	// IsAdmin reports whether the principal holds an admin role.
	IsAdmin(ctx context.Context) bool

	// IsSuperAdmin reports whether the principal holds a super-admin role.
	IsSuperAdmin(ctx context.Context) bool

	// Roles returns the roles assigned to this principal.
	Roles(ctx context.Context) []string

	// Groups returns the external groups this principal belongs to (e.g. the
	// "groups" claim from an OIDC JWT). Role membership is typically derived
	// from these groups via a role-definition mapping.
	Groups(ctx context.Context) []string

	// Expiry returns the time at which this principal's credentials expire.
	Expiry(ctx context.Context) time.Time
}
