package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/TouchBistro/gotham/cache"
	log "github.com/sirupsen/logrus"
)

// AllowAdminOnlyHttpMiddleware returns a net/http middleware function that creates a
// http.Handler wrapper to only allow "admin" or "super-admin" users through
func AllowAdminOnlyHttpMiddleware() Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			var ok bool
			var pr Principal

			_pr, err := getValue(r.Context(), ContextKeyPrincipal)

			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, err.Error())
				return
			}

			if _pr == nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
				return
			}
			if pr, ok = _pr.(Principal); !ok {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
				return
			}

			reqUserIsAdmin := pr.IsAdmin || pr.IsSuperAdmin
			if !reqUserIsAdmin {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make this request as it is not an administrator user", pr.Alias))
				return
			}

			// go to the next handler
			next.ServeHTTP(w, r)
		})
	})
}

// AllowAdminOrAliasGinHandler returns a net/http middleware that checks if the user alias
// found in the http request path segement tagged "pathParmName" is either the same
// as request context principal `alias` or the request context principal is an Admin/
// Super Admin; if not the request is aborted from further processing with an HTTP 401
// Unauthorized status code
//
// when matching /path/to/req/{id}, the value of "id" path parameter is matched to the principal
// alias
func AllowAdminOrAliasHttpMiddleware(pathParmName string) Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			var ok bool
			var pr Principal

			_pr, err := getValue(r.Context(), ContextKeyPrincipal)

			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, err.Error())
				return
			}

			if _pr == nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
				return
			}
			if pr, ok = _pr.(Principal); !ok {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
				return
			}

			userFromAuth := pr.Alias
			reqUserIsAdmin := pr.IsAdmin || pr.IsSuperAdmin
			userFromRequest := r.PathValue(pathParmName) // /path/to/resource/{x}

			if !reqUserIsAdmin && userFromAuth != userFromRequest {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make a request on behalf of %q", userFromAuth, userFromRequest))
				return
			}

			// go to the next handler
			next.ServeHTTP(w, r)
		})
	})
}

// AwsalbAuthorizeHttpMiddlewares returns an array of http middlewares as defind by the supplied
// auth policy. The pre & post actions are converted to a handler fn, that run before & after the
// main policy handler. The policy items are used by the main hanlder to match the incoming request
// against the claims & policy statements in order of definitiob to decide if the request must be
// allowed, or aborted.
// The
func AwsalbAuthorizeHttpMiddlewares(pol AuthPolicy, loader PrincipalLoader) []Middleware {
	middlewares := make([]Middleware, 0)
	middlewares = append(middlewares, actionProcessingHttpMiddlewares(pol.PreActions)...)
	middlewares = append(middlewares, awsalbAuthHttpMiddleware(pol, loader))
	middlewares = append(middlewares, actionProcessingHttpMiddlewares(pol.PreActions)...)
	return middlewares
}

// helper function

// actionProcessingHttpMiddlewares creates net/http middleware functions for the supplied
// policy actions list. 1 handler per defined action is returns
func actionProcessingHttpMiddlewares(actions []PolicyAction) []Middleware {
	funcs := make([]Middleware, 0)
	for _, action := range actions {
		funcs = append(funcs, action.toHttpMiddlewareFunc())
	}
	return funcs
}

// awsalbAuthHttpMiddleware returns an net/http middleware that uses the supplied auth policy
// & the JWT-encoded oidc user claims from the supplied http request header & decides
// if the request must be processed further or aborted
func awsalbAuthHttpMiddleware(ap AuthPolicy, loader PrincipalLoader) Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			log.Debugf("processing auth for %v %v", r.Method, r.URL.Path)

			// ensure the request is upgraded with a value map
			r = upgradeRequestContext(r)
			ctx := r.Context()

			var err error

			var sub string
			if sub, err = httpRequestHeaderValue(r, ap.Config.JwtConfig.SubClaimHeader, 0); err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "no sub claim value found from header")
				return
			}

			// fetch cached principal
			var pr *Principal
			prefix := "principal"
			cache, err := cache.Initialize()
			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "error initialzing cache")
				return
			}

			cloader := CachePrincipalLoader{prefix, cache}
			if pr, err = cloader.FetchPrincipal(ctx, sub); err != nil {

				var oidcDataHeaderVal string
				if sub, err = httpRequestHeaderValue(r, ap.Config.JwtConfig.IdTokenHeader, 0); err != nil {
					abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "no id token value found from header")
					return
				}

				jloader := JwtClaimsPrincipalLoader{
					config: ap.Config,
					jwt:    oidcDataHeaderVal,
				}
				if pr, err = jloader.FetchPrincipal(ctx, sub); err != nil {
					abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "error loading principal from cliams in JWT")
					return
				}

				// if login claim isn't there, we need to fill/sync it up from the supplied principal loader
				// this is suppose to fetch a Principal from a system of record like a DB or some other application
				// specific store
				if pr.Login == "" {
					// var prFromDb *Principal
					prFromDb := pr // init with the item from cache
					if loader != nil {
						if prFromDb, err = loader.FetchPrincipal(ctx, sub); err != nil {
							// if prFromDb, err = loadPrincipalFromDb(ctx, ap.Config, sub); err != nil {
							abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, fmt.Sprintf("principal JWT token didn't contain enough claims, but error fetching principal auth info from database/n%v", err.Error()))
							return
						}
					}

					// here we fill out roles from the gruops that are policy def specific
					prFromDb.Roles, prFromDb.IsSuperAdmin, prFromDb.IsAdmin = rolesFromGroups(ap.Config, prFromDb.Groups)

					// merge the principal from cliams with the principal from storage
					pr.Merge(*prFromDb)                           // merge with the principal obj from database
					pr.Expiry = time.Now().Add(119 * time.Second) // force 2m expiry after merge to eff ignore setting expiry from the database record
				}

				// put raw token in the principal obj context
				pr.RawToken = oidcDataHeaderVal

				// before caching, we force the expiry in principal to 2 min
				if err = cloader.Persist(ctx, *pr); err != nil {
					log.Warnf("error caching principal for external id %v", sub)
				}
			}

			pol, err := ap.AuthrPolicies.Match(*pr, *r)
			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, err.Error())
				return
			}

			if pol.Effect != PolicyEffectAllow {
				msg := fmt.Sprintf("access to %v %v to %v denied by auth policy", r.Method, r.URL, pr.Login)
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, msg)
				return
			}

			// set principal to context, all set go to next handler...
			if err = setValue(ctx, ContextKeyPrincipal, *pr); err != nil {
				msg := fmt.Sprintf("access to %v %v to %v denied, error saving principal in context due to %v", r.Method, r.URL, pr.Login, err.Error())
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, msg)
				return
			}
			if err = setValue(ctx, ContextKeyAlias, pr.Alias); err != nil {
				msg := fmt.Sprintf("access to %v %v to %v denied, error saving principal in context due to %v", r.Method, r.URL, pr.Login, err.Error())
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, msg)
				return
			}

			next.ServeHTTP(w, r)
		})
	})
}

// abortRespondAndLogErrorHttp abrts processing of http hanlder, sends an http response with
// the supplied message, http status code & a failure response code
func abortRespondAndLogErrorHttp(w http.ResponseWriter, r *http.Request, httpStatusCode int, msg string) {
	log.Error(msg)

	bytes := []byte(msg)

	resp := ResponseEnvelop{
		Request: r.URL.Path,
		Data:    msg,
		Code:    1, //TODO: define constant for this
	}

	bytes2, err := json.Marshal(resp)
	if err != nil {
		log.Error()
	} else {
		bytes = bytes2
	}

	// set response
	w.WriteHeader(httpStatusCode)
	w.Write(bytes)
}
