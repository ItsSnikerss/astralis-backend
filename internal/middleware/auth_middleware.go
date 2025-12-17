// internal/middleware/auth_middleware.go
package middleware

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

type contextKey string
const userContextKey = contextKey("userClaims")

var jwtKey = []byte("your_very_secret_key_that_is_long_and_secure")

// Наша кастомная структура claims, должна быть такой же как в auth_handler
type AppClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			http.Error(w, `{"error":"Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}
		tokenString := headerParts[1]

		// Парсим токен с использованием нашей новой структуры AppClaims
		claims := &AppClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Теперь функция возвращает нашу кастомную структуру
func GetClaimsFromContext(r *http.Request) (*AppClaims, bool) {
	claims, ok := r.Context().Value(userContextKey).(*AppClaims)
	return claims, ok
}