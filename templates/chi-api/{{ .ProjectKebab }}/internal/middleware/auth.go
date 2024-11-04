package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func NewAuthMiddleware(tokenPath string) (func(http.Handler) http.Handler, error) {
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token at %s: %w", tokenPath, err)
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			tokenFromHeader := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenFromHeader == strings.TrimSuffix(string(token), "\n") {
				next.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
		}
		return http.HandlerFunc(fn)
	}, nil
}
