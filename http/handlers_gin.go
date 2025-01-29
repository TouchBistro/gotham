package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/TouchBistro/gotham/cache"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// AllowAdminOnlyGinHandler returns a gin handler that checks if the request
// context principal is an Admin /Super Admin user; if not the request is aborted
// from further processing with an HTTP 401 Unauthorized status code
func AllowAdminOnlyGinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {

		var pr Principal
		if v, ok := c.Get(ContextKeyPrincipal); !ok {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		} else if pr, ok = v.(Principal); !ok {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		}

		reqUserIsAdmin := pr.IsAdmin || pr.IsSuperAdmin
		if !reqUserIsAdmin {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make as it is not an administrator user", pr.Alias))
			return
		}
	}
}

// AllowAdminOrAliasGinHandler returns a gin handler that checks if the user alias
// found in the http request path segement tagged "pathParmName" is either the same
// as request context principal `alias` or the request context principal is an Admin/
// Super Admin; if not the request is aborted from further processing with an HTTP 401
// Unauthorized status code
func AllowAdminOrAliasGinHandler(pathParmName string) gin.HandlerFunc {
	return func(c *gin.Context) {

		var pr Principal
		if v, ok := c.Get(ContextKeyPrincipal); !ok {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		} else if pr, ok = v.(Principal); !ok {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		}

		userFromAuth := pr.Alias
		reqUserIsAdmin := pr.IsAdmin || pr.IsSuperAdmin
		userFromRequest := c.Param(pathParmName)

		if !reqUserIsAdmin && userFromAuth != userFromRequest {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make a request on behalf of %q", userFromAuth, userFromRequest))
			return
		}
	}
}

// AwsalbAuthorizeGinHandler returns an array of gin hanlders as defind by the supplied
// auth policy. The pre & post actions are converted to a handler fn, that run before & after the
// main policy handler. The policy items are used by the main hanlder to match the incoming request
// against the claims & policy statements in order of definitiob to decide if the request must be
// allowed, or aborted.
// The
func AwsalbAuthorizeGinHandler(pol AuthPolicy, loader PrincipalLoader) []gin.HandlerFunc {
	funcs := make([]gin.HandlerFunc, 0)
	funcs = append(funcs, actionProcessingGinHandler(pol.PreActions)...)
	funcs = append(funcs, awsalbAuthGinHandler(pol, loader))
	funcs = append(funcs, actionProcessingGinHandler(pol.PostActions)...)
	return funcs
}

// helper function

// actionProcessingGinHandler creates gin middleware functions for the supplied
// policy actions list. 1 handler per defined action is returns
func actionProcessingGinHandler(actions []PolicyAction) []gin.HandlerFunc {
	funcs := make([]gin.HandlerFunc, 0)
	for _, action := range actions {
		funcs = append(funcs, action.toGinHandler())
	}
	return funcs
}

// GetJwtAuthMiddleware returns a gin middleware that uses the supplied auth policy
// & the JWT-encoded oidc user claims from the supplied http request header & decides
// if the request must be processed further or aborted
func awsalbAuthGinHandler(ap AuthPolicy, loader PrincipalLoader) gin.HandlerFunc {
	return func(c *gin.Context) {

		log.Debugf("processing auth for %v %v", c.Request.Method, c.Request.URL.Path)

		ctx := c.Request.Context()

		var err error

		var sub string
		if sub, err = httpRequestHeaderValue(c.Request, ap.Config.JwtConfig.SubClaimHeader, 0); err != nil {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "no sub claim value found from header")
			return
		}

		// fetch cached principal
		var pr *Principal
		prefix := "principal"
		cache, err := cache.Initialize()
		if err != nil {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "error initialzing cache")
			return
		}

		cloader := CachePrincipalLoader{prefix, cache}
		if pr, err = cloader.FetchPrincipal(ctx, sub); err != nil {

			var oidcDataHeaderVal string
			if sub, err = httpRequestHeaderValue(c.Request, ap.Config.JwtConfig.IdTokenHeader, 0); err != nil {
				abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "no id token value found from header")
				return
			}

			jloader := JwtClaimsPrincipalLoader{
				config: ap.Config,
				jwt:    oidcDataHeaderVal,
			}
			if pr, err = jloader.FetchPrincipal(ctx, sub); err != nil {
				abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "error loading principal from cliams in JWT")
				return
			}

			// if login claim isn't there, we need to fill/sync it up from the supplied principal loader
			// this is suppose to fetch a Principal from a system of record like a DB or some other application
			// specific store
			if pr.Login == "" {
				var prFromDb *Principal
				if prFromDb, err = loader.FetchPrincipal(ctx, sub); err != nil {
					// if prFromDb, err = loadPrincipalFromDb(ctx, ap.Config, sub); err != nil {
					abortRespondAndLogErrorGin(c, http.StatusUnauthorized, fmt.Sprintf("principal JWT token didn't contain enough claims, but error fetching principal auth info from database/n%v", err.Error()))
					return
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

		pol, err := ap.AuthrPolicies.Match(*pr, *c.Request)
		if err != nil {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, err.Error())
			return
		}

		if pol.Effect != PolicyEffectAllow {
			msg := fmt.Sprintf("access to %v %v to %v denied by auth policy", c.Request.Method, c.Request.URL, pr.Login)
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, msg)
			return
		}

		// set principal to context, all set go to next handler...
		c.Set(ContextKeyPrincipal, *pr)  // set pr for later use
		c.Set(ContextKeyAlias, pr.Alias) // set alias for each fetch
	}
}

// abortRespondAndLogErrorGin aborts processing of gin hanlder, sends an http response with
// the supplied message, http status code & a failure response code
func abortRespondAndLogErrorGin(c *gin.Context, httpStatusCode int, msg string) {
	log.Error(msg)
	c.AsciiJSON(httpStatusCode, ResponseEnvelop{
		Request: c.Request.URL.Path,
		Data:    msg,
		Code:    1, //TODO: define constant for this
	})
	c.Abort()
}
