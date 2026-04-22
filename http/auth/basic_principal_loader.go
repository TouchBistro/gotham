package auth

import (
	"context"
)

// BasicPrincipalLoader returns a PrincipalLoader that always yields a copy of
// the supplied BasicPrincipal, with its Id overridden to the incoming subject.
// Useful for tests and for bootstrapping known principals without a backing
// token or cache.
func BasicPrincipalLoader(pr BasicPrincipal) PrincipalLoaderFunc {
	return func(ctx context.Context, in FetchPrincipalInput) (FetchPrincipalOutput, error) {
		p := pr
		p.Id = in.Id
		return FetchPrincipalOutput{Principal: &p}, nil
	}
}
