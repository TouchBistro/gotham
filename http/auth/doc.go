// Package auth defines an interface-based authentication and authorization
// model for HTTP request pipelines.
//
// # Overview
//
// The package is built around three pieces:
//
//   - Principal — an interface abstracting the authenticated subject. All
//     accessors are context-scoped: Identifier, IsAdmin, IsSuperAdmin, Roles,
//     Groups, Expiry.
//   - PrincipalLoader — produces a Principal for an incoming request. Loaders
//     take a FetchPrincipalInput (Id, Request, PolicyConfig) and return a
//     FetchPrincipalOutput.
//   - Config — the authorization policy: role definitions, admin/super-admin
//     role sets, pre/post Actions, and an ordered Policies list.
//
// Consumers assemble a Config, pick a PrincipalLoader, and hand both to the
// middleware constructors in the parent http package.
//
// # Principal implementations
//
// BasicPrincipal is the canonical concrete implementation — a struct with
// Id, RoleSet, Admin, SuperAdmin, ExpiryAt, Grps. It is also the default
// decode target for CachePrincipalLoader and the seed for BasicPrincipalLoader.
//
// JwtPrincipal embeds BasicPrincipal and additionally carries the Raw token
// and parsed Claims map. It is produced by JwtPrincipalLoader.
//
// Consumers may supply their own Principal implementation; the interface is
// the only contract.
//
// # Loaders
//
// Three loaders are provided:
//
//   - BasicPrincipalLoader(seed BasicPrincipal) — returns a copy of seed with
//     Id overridden to the incoming FetchPrincipalInput.Id. Intended for
//     tests and bootstrap scenarios.
//
//   - JwtPrincipalLoader — reads a JWT token from a configured HTTP header on
//     the incoming request, parses its claims, and lifts the configured sub
//     and expiry claims onto a *JwtPrincipal. Constructed via the builder:
//
//     loader := auth.NewJwtPrincipalLoader().
//     WithHttpHeaderName("x-jwt-data").
//     WithSubClaimName("sub").        // optional, defaults to "sub"
//     WithExpiryClaimName("exp").     // optional, defaults to "exp"
//     Build()
//
//   - CachePrincipalLoader[T Principal] — backed by a cache.MemoryCache.
//     Generic over the concrete Principal type T to guarantee correct decode
//     targets. Use the same T for Persist and FetchPrincipal — typically a
//     pointer type such as *BasicPrincipal:
//
//     loader := auth.CachePrincipalLoader[*auth.BasicPrincipal]{
//     KeyPrefix: "principals",
//     Cache:     memCache,
//     }
//     _ = loader.Persist(ctx, pr)
//     out, err := loader.FetchPrincipal(ctx, auth.FetchPrincipalInput{Id: "alice"})
//
// Custom loaders implement PrincipalLoader directly, or wrap an ordinary
// function with PrincipalLoaderFunc.
//
// # Config and policy matching
//
// Config bundles role definitions, admin/super-admin role sets, pre/post
// Actions, and an ordered Policies list. Load one from disk via
// LoadConfigFromFile (falls back to a safe default on error).
//
// Config.RolesForPrincipal(ctx, p) resolves the set of roles a principal is
// entitled to, and reports admin/super-admin status. Matching sources:
//   - Any of the principal's groups appears in a role's member set.
//   - The role's member set contains the Wildcard sentinel "*".
//   - The role's member set contains the per-user token "user(<id>)",
//     where <id> is the principal's Identifier.
//
// Policies.Match(ctx, p, req) walks the ordered rule list and returns the
// first rule whose method, path, and subject set all match the principal and
// request. A returned Item with Effect == EffectAllow indicates the request
// is authorized; any other outcome (no match, error, EffectDeny) means the
// request should be denied by the caller.
//
// # Actions
//
// Action describes a pre- or post-processing step on the incoming request
// (e.g. setting or removing a header via reflection). Pre-actions run before
// principal loading; post-actions run after a successful policy match. They
// are attached on Config via PreActions / PostActions and wired by the
// middleware constructors.
//
// # Typical wiring
//
// Consumers do not normally call Match, RolesForPrincipal, or the loader
// directly — those are invoked by the middleware constructors in the parent
// http package (AwsalbAuthorizeGinHandler / AwsalbAuthorizeHttpMiddlewares).
// See package http for end-to-end wiring examples.
package auth
