package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/TouchBistro/gotham/http/auth"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// AllowAdminOnlyGinHandler returns a gin handler that checks if the request
// context principal is an Admin /Super Admin user; if not the request is aborted
// from further processing with an HTTP 401 Unauthorized status code
func AllowAdminOnlyGinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var pr auth.Principal
		if v, ok := c.Get(ContextKeyPrincipal); !ok {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		} else if pr, ok = v.(auth.Principal); !ok {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		}

		if !pr.IsAdmin(ctx) && !pr.IsSuperAdmin(ctx) {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make as it is not an administrator user", pr.Identifier(ctx)))
			return
		}
	}
}

// AllowAdminOrAliasGinHandler returns a gin handler that checks if the user alias
// found in the http request path segement tagged "pathParmName" is either the same
// as request context principal identifier or the request context principal is an
// Admin/Super Admin; if not the request is aborted from further processing with an
// HTTP 401 Unauthorized status code
func AllowAdminOrAliasGinHandler(pathParmName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var pr auth.Principal
		if v, ok := c.Get(ContextKeyPrincipal); !ok {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		} else if pr, ok = v.(auth.Principal); !ok {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		}

		userFromAuth := pr.Identifier(ctx)
		reqUserIsAdmin := pr.IsAdmin(ctx) || pr.IsSuperAdmin(ctx)
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
// against the claims & policy statements in order of definition to decide if the request must be
// allowed, or aborted. The subHeader parameter represents the http request header that holds
// the value for the sub claim, that is used to lookup a loader.
func AwsalbAuthorizeGinHandler(pol auth.Config, subHeader string, loader auth.PrincipalLoader) []gin.HandlerFunc {
	funcs := make([]gin.HandlerFunc, 0)
	funcs = append(funcs, actionProcessingGinHandler(pol.PreActions)...)
	funcs = append(funcs, awsalbAuthGinHandler(pol, subHeader, loader))
	funcs = append(funcs, actionProcessingGinHandler(pol.PostActions)...)
	return funcs
}

// helper function

// actionProcessingGinHandler creates gin middleware functions for the supplied
// policy actions list. 1 handler per defined action is returned.
func actionProcessingGinHandler(actions []auth.Action) []gin.HandlerFunc {
	funcs := make([]gin.HandlerFunc, 0, len(actions))
	for _, action := range actions {
		funcs = append(funcs, func(c *gin.Context) {
			_ = action.Apply(c.Request)
		})
	}
	return funcs
}

// awsalbAuthGinHandler returns a minimal gin middleware that performs basic
// principal-loading auth followed by a policy match: retrieve context, read
// the "sub" request header, use the supplied PrincipalLoader to fetch the
// principal, verify the principal is not expired, then match the request
// against pol.AuthrPolicies. On any failure (load error, expired principal,
// match error, non-Allow effect) the request is aborted with HTTP 401.
// On success the principal is stored on the gin context and the chain
// proceeds.
func awsalbAuthGinHandler(pol auth.Config, subHeader string, loader auth.PrincipalLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("processing auth for %v %v", c.Request.Method, c.Request.URL.Path)

		ctx := c.Request.Context()

		sub, err := httpRequestHeaderValue(c.Request, subHeader, 0)
		if err != nil {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "no sub claim value found from header")
			return
		}

		out, err := loader.FetchPrincipal(ctx, auth.FetchPrincipalInput{
			Id:           sub,
			Request:      *c.Request,
			PolicyConfig: pol,
		})
		if err != nil {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, fmt.Sprintf("error loading principal: %v", err))
			return
		}
		pr := out.Principal

		exp := pr.Expiry(ctx)
		if !exp.IsZero() && exp.Before(time.Now()) {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, "principal has expired")
			return
		}

		matched, err := pol.AuthrPolicies.Match(ctx, pr, *c.Request)
		if err != nil {
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, err.Error())
			return
		}
		if matched.Effect != auth.EffectAllow {
			msg := fmt.Sprintf("access to %v %v to %v denied by auth policy", c.Request.Method, c.Request.URL, pr.Identifier(ctx))
			abortRespondAndLogErrorGin(c, http.StatusUnauthorized, msg)
			return
		}

		c.Set(ContextKeyPrincipal, pr)
		c.Set(ContextKeyAlias, pr.Identifier(ctx))
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
