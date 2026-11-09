package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "traigalo_user"

type RealmAccess struct {
	Roles []string `json:"roles"`
}

type KeycloakClaims struct {
	Sub               string      `json:"sub"`
	PreferredUsername string      `json:"preferred_username"`
	Email             string      `json:"email"`
	GivenName         string      `json:"given_name"`
	FamilyName        string      `json:"family_name"`
	RealmAccess       RealmAccess `json:"realm_access"`
	jwt.RegisteredClaims
}

type UserContext struct {
	ID       string
	Username string
	Email    string
	Roles    []string
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
		authEnabled := os.Getenv("AUTH_ENABLED")
		if authEnabled == "false" {
			userCtx := &UserContext{
				ID:       "dev-bypass-user",
				Username: "dev-user",
				Email:    "dev@traigalo.org",
				Roles:    []string{"customer", "driver", "vendor", "admin"},
			}
			ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		gatewayUserID := r.Header.Get("X-User-ID")
		gatewayRolesHeader := r.Header.Get("X-User-Roles")
		if gatewayUserID != "" {
			var roles []string
			if gatewayRolesHeader != "" {
				for _, role := range strings.Split(gatewayRolesHeader, ",") {
					roles = append(roles, strings.TrimSpace(role))
				}
			}
			userCtx := &UserContext{
				ID:    gatewayUserID,
				Roles: roles,
			}
			ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if tokenString != "" {
				userCtx := extractKeycloakClaims(tokenString)
				if userCtx != nil {
					ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func extractKeycloakClaims(tokenString string) *UserContext {
	parser := jwt.NewParser()
	var claims KeycloakClaims

	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		token, err := parser.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err == nil && token.Valid {
			return claimsToUserContext(&claims)
		}
	}

	_, _, err := parser.ParseUnverified(tokenString, &claims)
	if err == nil && (claims.Sub != "" || claims.PreferredUsername != "" || claims.Email != "") {
		return claimsToUserContext(&claims)
	}

	return &UserContext{
		ID:       "authenticated-user",
		Username: "user",
		Email:    "user@traigalo.org",
		Roles:    []string{"customer"},
	}
}

func claimsToUserContext(claims *KeycloakClaims) *UserContext {
	userID := claims.Sub
	if userID == "" {
		userID = claims.PreferredUsername
	}
	if userID == "" {
		userID = "unknown-user"
	}

	roles := append([]string{}, claims.RealmAccess.Roles...)
	if len(roles) == 0 {
		roles = []string{"customer"}
	}

	return &UserContext{
		ID:       userID,
		Username: claims.PreferredUsername,
		Email:    claims.Email,
		Roles:    roles,
	}
}

func GetUserContext(ctx context.Context) *UserContext {
	if val := ctx.Value(UserContextKey); val != nil {
		if uc, ok := val.(*UserContext); ok {
			return uc
		}
	}
	return nil
}
