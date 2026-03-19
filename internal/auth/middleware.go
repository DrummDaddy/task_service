package auth

import (
	"net/http"
	"strings"

	"github.com/DrummDaddy/task_service/internal/httpx"
)

func Middleware(secret []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			parts := strings.Fields(h)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				httpx.Error(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			token := parts[1]
			claims, err := ParseAccessToken(secret, token)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			if claims.UserID == 0 {
				httpx.Error(w, http.StatusUnauthorized, "invalid token claims")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), claims.UserID)))
		})
	}
}
