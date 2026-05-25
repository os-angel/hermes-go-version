package restapi

import (
	"context"
	"net/http"
	"strings"
)

// authKey es la clave del contexto para el API key autenticado.
type authKey struct{}

// BearerAuth es un middleware que valida el header Authorization: Bearer <token>.
func BearerAuth(validTokens []string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(validTokens))
	for _, t := range validTokens {
		set[t] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r.Header.Get("Authorization"))
			if token == "" {
				http.Error(w, "missing authorization", http.StatusUnauthorized)
				return
			}
			if _, ok := set[token]; !ok {
				http.Error(w, "invalid token", http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), authKey{}, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}
