package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/TouchBistro/gotham/http/auth"
	log "github.com/sirupsen/logrus"
)

// AllowAdminOnlyHttpMiddleware returns a net/http middleware function that creates a
// http.Handler wrapper to only allow "admin" or "super-admin" users through
func AllowAdminOnlyHttpMiddleware() Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			_pr, err := getValue(ctx, ContextKeyPrincipal)
			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, err.Error())
				return
			}
			if _pr == nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
				return
			}
			pr, ok := _pr.(auth.Principal)
			if !ok {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
				return
			}

			if !pr.IsAdmin(ctx) && !pr.IsSuperAdmin(ctx) {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make this request as it is not an administrator user", pr.Identifier(ctx)))
				return
			}

			next.ServeHTTP(w, r)
		})
	})
}

// AllowAdminOrAliasHttpMiddleware returns a net/http middleware that checks if the
// user identifier found in the http request path segment tagged "pathParmName" is
// either the same as the request context principal's identifier, or the request
// context principal is an Admin / Super Admin; if not the request is aborted from
// further processing with an HTTP 401 Unauthorized status code.
//
// when matching /path/to/req/{id}, the value of "id" path parameter is matched to
// the principal identifier.
func AllowAdminOrAliasHttpMiddleware(pathParmName string) Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			_pr, err := getValue(ctx, ContextKeyPrincipal)
			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, err.Error())
				return
			}
			if _pr == nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
				return
			}
			pr, ok := _pr.(auth.Principal)
			if !ok {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
				return
			}

			userFromAuth := pr.Identifier(ctx)
			reqUserIsAdmin := pr.IsAdmin(ctx) || pr.IsSuperAdmin(ctx)
			userFromRequest := r.PathValue(pathParmName)

			if !reqUserIsAdmin && userFromAuth != userFromRequest {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make a request on behalf of %q", userFromAuth, userFromRequest))
				return
			}

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
func AwsalbAuthorizeHttpMiddlewares(pol auth.Config, loader auth.PrincipalLoader) []Middleware {
	middlewares := make([]Middleware, 0)
	middlewares = append(middlewares, actionProcessingHttpMiddlewares(pol.PreActions)...)
	middlewares = append(middlewares, awsalbAuthHttpMiddleware(pol, loader))
	middlewares = append(middlewares, actionProcessingHttpMiddlewares(pol.PostActions)...)
	return middlewares
}

// helper function

// actionProcessingHttpMiddlewares creates net/http middleware functions for the supplied
// policy actions list. 1 middleware per defined action is returned.
func actionProcessingHttpMiddlewares(actions []auth.Action) []Middleware {
	funcs := make([]Middleware, 0, len(actions))
	for _, action := range actions {
		funcs = append(funcs, MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = action.Apply(r)
				next.ServeHTTP(w, r)
			})
		}))
	}
	return funcs
}

// awsalbAuthHttpMiddleware returns a minimal net/http middleware that performs
// basic principal-loading auth followed by a policy match: ensure the request
// context has a value map, read the "sub" request header, use the supplied
// PrincipalLoader to fetch the principal, verify the principal is not expired,
// then match the request against pol.AuthrPolicies. On any failure (load
// error, expired principal, match error, non-Allow effect) the request is
// aborted with HTTP 401. On success the principal + alias are stored on the
// request context and the chain proceeds.
func awsalbAuthHttpMiddleware(pol auth.Config, loader auth.PrincipalLoader) Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Debugf("processing auth for %v %v", r.Method, r.URL.Path)

			r = upgradeRequestContext(r)
			ctx := r.Context()

			sub, err := httpRequestHeaderValue(r, "sub", 0)
			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "no sub claim value found from header")
				return
			}

			out, err := loader.FetchPrincipal(ctx, auth.FetchPrincipalInput{
				Id:           sub,
				Request:      *r,
				PolicyConfig: pol,
			})
			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, fmt.Sprintf("error loading principal: %v", err))
				return
			}
			pr := out.Principal

			exp := pr.Expiry(ctx)
			if !exp.IsZero() && exp.Before(time.Now()) {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, "principal has expired")
				return
			}

			matched, err := pol.AuthrPolicies.Match(ctx, pr, *r)
			if err != nil {
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, err.Error())
				return
			}
			if matched.Effect != auth.EffectAllow {
				msg := fmt.Sprintf("access to %v %v to %v denied by auth policy", r.Method, r.URL, pr.Identifier(ctx))
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, msg)
				return
			}

			if err = setValue(ctx, ContextKeyPrincipal, pr); err != nil {
				msg := fmt.Sprintf("access to %v %v to %v denied, error saving principal in context due to %v", r.Method, r.URL, pr.Identifier(ctx), err.Error())
				abortRespondAndLogErrorHttp(w, r, http.StatusUnauthorized, msg)
				return
			}
			if err = setValue(ctx, ContextKeyAlias, pr.Identifier(ctx)); err != nil {
				msg := fmt.Sprintf("access to %v %v to %v denied, error saving principal in context due to %v", r.Method, r.URL, pr.Identifier(ctx), err.Error())
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
	_, _ = w.Write(bytes)
}
