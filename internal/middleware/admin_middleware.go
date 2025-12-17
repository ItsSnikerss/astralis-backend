package middleware

import (
	"net/http"
)

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaimsFromContext(r)
		if !ok {
			http.Error(w, `{"error":"Could not retrieve user claims"}`, http.StatusInternalServerError)
			return
		}

		if claims.Role != "admin" {
			http.Error(w, `{"error":"Forbidden: Administrator access required"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}