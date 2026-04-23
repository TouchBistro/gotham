// Package http provides authentication / authorization middlewares and small
// helpers for both net/http and the gin-gonic framework. Principal and policy
// primitives live in the sub-package http/auth; this package wires those into
// request pipelines and exposes role-gated convenience handlers.
//
// # Middleware constructors
//
// Two entry points build the full auth chain (pre-actions → auth → post-actions)
// from an auth.Config and an auth.PrincipalLoader:
//
//   - AwsalbAuthorizeGinHandler(pol auth.Config, subHeader string, loader auth.PrincipalLoader) []gin.HandlerFunc
//   - AwsalbAuthorizeHttpMiddlewares(pol auth.Config, loader auth.PrincipalLoader) []Middleware
//
// The core auth step:
//  1. Reads the subject header (gin: subHeader parameter; net/http: "sub").
//  2. Calls loader.FetchPrincipal with the subject, request, and policy config.
//  3. Rejects with 401 on load error, expired principal, no matching rule,
//     or a rule with Effect != auth.EffectAllow.
//  4. On success, stores the Principal under ContextKeyPrincipal and the
//     principal's Identifier under ContextKeyAlias, then proceeds.
//
// # Role-gated handlers
//
// After the auth middleware has populated the context, these helpers gate
// downstream handlers on the principal's admin/identity state:
//
//   - AllowAdminOnlyGinHandler / AllowAdminOnlyHttpMiddleware — allow only
//     principals where IsAdmin or IsSuperAdmin is true.
//   - AllowAdminOrAliasGinHandler(pathParmName) /
//     AllowAdminOrAliasHttpMiddleware(pathParmName) — allow admins, or a
//     principal whose Identifier matches the named path parameter.
//
// # Context access
//
// The net/http side stores per-request values in a map attached to the
// request context via upgradeRequestContext. setValue / getValue read and
// write that map. The gin side uses c.Set / c.Get directly.
//
// Downstream handlers read the authenticated principal as follows:
//
//	// gin
//	if v, ok := c.Get(http.ContextKeyPrincipal); ok {
//	    pr := v.(auth.Principal)
//	    _ = pr.Identifier(c.Request.Context())
//	}
//
//	// net/http
//	v, _ := http.GetContextPrincipal(r.Context()) // or getValue + type assert
//
// # Typical wiring
//
//	cfg, _ := auth.LoadConfigFromFile("policy.json")
//	loader := auth.NewJwtPrincipalLoader().WithHttpHeaderName("x-jwt-data").Build()
//
//	// gin
//	r := gin.New()
//	r.Use(http.AwsalbAuthorizeGinHandler(*cfg, "sub", loader)...)
//	r.GET("/admin/:user", http.AllowAdminOrAliasGinHandler("user"), adminHandler)
//
//	// net/http
//	var h stdhttp.Handler = mux
//	for _, m := range http.AwsalbAuthorizeHttpMiddlewares(*cfg, loader) {
//	    h = m.Wrap(h)
//	}
//
// # Other exports
//
//   - Middleware / MiddlewareFunc — the net/http middleware contract.
//   - ResponseEnvelop — the JSON envelope used by the 401 responses emitted
//     by the auth middlewares.
//   - ContextKeyPrincipal, ContextKeyAlias, ContextMapKey — context keys.
package http
