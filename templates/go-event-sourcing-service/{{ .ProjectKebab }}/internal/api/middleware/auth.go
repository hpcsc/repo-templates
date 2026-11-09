package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/response"
)

type userContextKey struct{}

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func UserFromContext(ctx context.Context) string {
	user, _ := ctx.Value(userContextKey{}).(string)
	return user
}

func NewAuthMiddleware(tokenPath string) (func(http.Handler) http.Handler, error) {
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token at %s: %w", tokenPath, err)
	}
	expected := strings.TrimSuffix(string(token), "\n")

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			tokenFromHeader := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenFromHeader != expected {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(response.Fail("sign in to continue"))
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}, nil
}