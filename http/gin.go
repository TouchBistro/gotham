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
			abortRespondAndLogError(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		} else if pr, ok = v.(Principal); !ok {
			abortRespondAndLogError(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		}

		reqUserIsAdmin := pr.IsAdmin || pr.IsSuperAdmin
		if !reqUserIsAdmin {
			abortRespondAndLogError(c, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make as it is not an administrator user", pr.Alias))
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
			abortRespondAndLogError(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		} else if pr, ok = v.(Principal); !ok {
			abortRespondAndLogError(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		}

		userFromAuth := pr.Alias
		reqUserIsAdmin := pr.IsAdmin || pr.IsSuperAdmin
		userFromRequest := c.Param(pathParmName)

		if !reqUserIsAdmin && userFromAuth != userFromRequest {
			abortRespondAndLogError(c, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make a request on behalf of %q", userFromAuth, userFromRequest))
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

		ctx := c.Request.Context()
		log.Debugf("processing auth for %v %v", c.Request.Method, c.Request.URL.Path)

		subClaimHeader := ap.Config.JwtConfig.SubClaimHeader // get the sub claim header
		subClaimHeaderVals := c.Request.Header[subClaimHeader]
		if len(subClaimHeaderVals) == 0 {
			abortRespondAndLogError(c, http.StatusUnauthorized, "no sub claim value found from header")
			return
		}
		sub := subClaimHeaderVals[0] // grab the first one

		// fetch sub/external id from the cliams
		// var sub string
		// if sub_any, ok := claims["sub"]; !ok {
		// 	abortRespondAndLogError(c, http.StatusUnauthorized, "invalid id/subject returned in IdP token")
		// 	return
		// } else {
		// 	sub = sub_any.(string) // TODO risky, assert properly; but that's what it should be...
		// }

		// fetch cached principal
		var err error
		var pr *Principal
		prefix := "principal"
		cache, err := cache.Initialize()
		if err != nil {
			abortRespondAndLogError(c, http.StatusUnauthorized, "error initialzing cache")
			return
		}

		principalFromCache := CachePrincipalLoader{prefix, cache}
		if pr, err = principalFromCache.FetchPrincipal(ctx, sub); err != nil {

			// if err, fetch principal from the claims found inside the id_token
			idTokenHeader := ap.Config.JwtConfig.IdTokenHeader // get the id_token header
			oidcDataHeaderVals := c.Request.Header[idTokenHeader]
			if len(oidcDataHeaderVals) == 0 {
				abortRespondAndLogError(c, http.StatusUnauthorized, "no id token value found from header")
				return
			}
			oidcDataHeaderVal := oidcDataHeaderVals[0]

			// fetch claims from the incoming jwt token
			// claims, err := util.ClaimsFromJwt(oidcDataHeaderVal)
			// if err != nil {
			// 	abortRespondAndLogError(c, http.StatusUnauthorized, err.Error())
			// 	return
			// }

			principalFromClaims := JwtClaimsPrincipalLoader{
				config: ap.Config,
				jwt:    oidcDataHeaderVal,
			}
			if pr, err = principalFromClaims.FetchPrincipal(ctx, sub); err != nil {
				abortRespondAndLogError(c, http.StatusUnauthorized, "error loading principal from cliams in JWT")
				return
			}

			// if login claim isn't there, we need to fill/sync it up from the supplied principal loader
			// this is suppose to fetch a Principal from a system of record like a DB or some other application
			// specific store
			if pr.Login == "" {
				var prFromDb *Principal
				if prFromDb, err = loader.FetchPrincipal(ctx, sub); err != nil {
					// if prFromDb, err = loadPrincipalFromDb(ctx, ap.Config, sub); err != nil {
					abortRespondAndLogError(c, http.StatusUnauthorized, fmt.Sprintf("principal JWT token didn't contain enough claims, but error fetching principal auth info from database/n%v", err.Error()))
					return
				}

				// here we fill out roles from the gruops that are policy def specific
				prFromDb.Roles, prFromDb.IsSuperAdmin, prFromDb.IsAdmin = rolesFromGroups(ap.Config, pr.Groups)

				// merge the principal from cliams with the principal from storage
				pr.Merge(*prFromDb)                           // merge with the principal obj from database
				pr.Expiry = time.Now().Add(119 * time.Second) // force 2m expiry after merge to eff ignore setting expiry from the database record
			}

			// put raw token in the principal obj context
			pr.RawToken = oidcDataHeaderVal

			// before caching, we force the expiry in principal to 2 min
			if err = principalFromCache.Persist(ctx, *pr); err != nil {
				log.Warnf("error caching principal for external id %v", sub)
			}
		}

		pol, err := ap.AuthrPolicies.Match(*pr, *c.Request)
		if err != nil {
			abortRespondAndLogError(c, http.StatusUnauthorized, err.Error())
			return
		}

		if pol.Effect != PolicyEffectAllow {
			msg := fmt.Sprintf("access to %v %v to %v denied by auth policy", c.Request.Method, c.Request.URL, pr.Login)
			abortRespondAndLogError(c, http.StatusUnauthorized, msg)
			return
		}

		// set principal to context, all set go to next handler...
		c.Set(ContextKeyPrincipal, *pr)  // set pr for later use
		c.Set(ContextKeyAlias, pr.Alias) // set alias for each fetch
	}
}

// abortRespondAndLogError aborts processing of gin context, sends an http response with
// the supplied message, http status code & devops-api response failure
func abortRespondAndLogError(c *gin.Context, httpStatusCode int, msg string) {
	log.Error(msg)
	c.AsciiJSON(httpStatusCode, ResponseEnvelop{
		Request: c.Request.URL.Path,
		Data:    msg,
		Code:    1, //TODO: define constant for this
	})
	c.Abort()
}

// // rolesFromGroups using mapping defined in config, this func
// // returns a RoleSet from the list of groups that these supplied groups
// // lie into; also return if any of these roles are superAdmins & Admins
// func rolesFromGroups(cfg Config, grps []string) (Set, bool, bool) {

// 	var issa, isa bool

// 	if grps == nil {
// 		return nil, issa, isa
// 	}

// 	roles := Set{}
// 	for roleName, membersGroupset := range cfg.Roles.Definitions {
// 		// if any of the groups are part of this role...
// 		if membersGroupset.Contains(grps...) {
// 			// add the role to this list
// 			roles.Insert(roleName)

// 			// check if this role is admin; mark user admin
// 			if _, ok := cfg.Roles.AdminRoles[roleName]; ok {
// 				isa = ok
// 			}

// 			// check if this role is super admin; mark user super admin
// 			if _, ok := cfg.Roles.SuperAdminRoles[roleName]; ok {
// 				issa = ok
// 			}
// 		}
// 	}
// 	return roles, issa, isa
// }

// returnFirstNonZero returns the first non-zero string from he args
func returnFirstNonZero(str1 string, strn ...string) string {
	strs := append([]string{str1}, strn...)
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return str1
}
