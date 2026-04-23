package auth

import (
	"context"
	"net/http"
)

// FetchPrincipalInput bundles the request-scoped inputs a PrincipalLoader may
// need to resolve a Principal. It is passed by value to FetchPrincipal so the
// loader surface can grow without breaking existing implementations.
type FetchPrincipalInput struct {
	// Id is the subject identifier being resolved (typically the "sub" JWT
	// claim or an equivalent external user id).
	Id string

	// Request is the inbound HTTP request associated with this lookup. Loaders
	// may read headers, the URL, or other request state as needed.
	Request http.Request

	// PolicyConfig is the active auth policy configuration (role definitions,
	// admin / super-admin role sets, etc.) a loader may consult when
	// constructing the Principal.
	PolicyConfig Config
}

// FetchPrincipalOutput wraps the result of a PrincipalLoader lookup. It is a
// struct rather than a bare Principal so additional fields (e.g. cache-hit
// metadata, freshness hints) can be added later without breaking callers.
type FetchPrincipalOutput struct {
	// Principal is the resolved principal.
	Principal Principal
}

// PrincipalLoader is the abstraction used to fetch a Principal for an incoming
// request. Implementations may be backed by a cache, a JWT token's claims, or
// any other source.
type PrincipalLoader interface {
	FetchPrincipal(ctx context.Context, in FetchPrincipalInput) (FetchPrincipalOutput, error)
}

// PrincipalLoaderFunc is an adapter that allows ordinary functions to be used
// as PrincipalLoader implementations.
type PrincipalLoaderFunc func(ctx context.Context, in FetchPrincipalInput) (FetchPrincipalOutput, error)

// FetchPrincipal implements PrincipalLoader.
func (f PrincipalLoaderFunc) FetchPrincipal(ctx context.Context, in FetchPrincipalInput) (FetchPrincipalOutput, error) {
	return f(ctx, in)
}
