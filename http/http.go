package http

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func GetAdminCheckingMiddlewareNetHttp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var pr Principal
		var ok bool

		_pr := r.Context().Value(ContextKeyPrincipal)
		if _pr == nil {
			abortRespondAndLogError2(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		}
		if pr, ok = _pr.(Principal); !ok {
			abortRespondAndLogError2(w, r, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
			return
		}

		_ = pr

		// if v, ok := c.Get(ContextKeyPrincipal); !ok {
		// } else if pr, ok = v.(types.Principal); !ok {
		// 	abortRespondAndLogError(c, http.StatusUnauthorized, "couldn't retrieve auth context from this request")
		// 	return
		// }

		// reqUserIsAdmin := pr.IsAdmin || pr.IsSuperAdmin
		// if !reqUserIsAdmin {
		// 	abortRespondAndLogError(c, http.StatusUnauthorized, fmt.Sprintf("%q not authorized to make as it is not an administrator user", pr.Alias))
		// 	return
		// }
	}
}

func abortRespondAndLogError2(w http.ResponseWriter, r *http.Request, httpStatusCode int, msg string) {
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
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(bytes)
}
