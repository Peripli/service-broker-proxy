package middleware

import (
	"net/http"

	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/httputils"
)

const notAuthorized = "Not Authorized"

func BasicAuth(username, password string) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authorized(r, username, password) {
				httputils.WriteResponse(w, http.StatusUnauthorized, httputils.HTTPErrorResponse{
					ErrorKey:     notAuthorized,
					ErrorMessage: fmt.Sprintf("User %s is not not authorized to access this resource", username),
				},
				)
				return
			}
			handler.ServeHTTP(w, r)
		})
	}
}

func authorized(r *http.Request, username, password string) bool {
	u, p, isOk := r.BasicAuth()
	return isOk && username == u && password == p
}
