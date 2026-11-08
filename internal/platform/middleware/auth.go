package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "traigalo_user"

type UserContext struct {
	ID    string
	Roles []string
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-User-ID, X-User-Roles")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func GatewayAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		rolesHeader := r.Header.Get("X-User-Roles")

		if userID == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				userID = "authenticated-dev-user"
				rolesHeader = "customer,driver,vendor,admin"
			}
		}

		if userID != "" {
			var roles []string
			if rolesHeader != "" {
				for _, role := range strings.Split(rolesHeader, ",") {
					roles = append(roles, strings.TrimSpace(role))
				}
			}
			ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
				ID:    userID,
				Roles: roles,
			})
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
