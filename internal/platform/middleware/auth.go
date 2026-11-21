package middleware

import (
	"context"
	"crypto/rsa"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserContextKey contextKey = "traigalo_user"
	clockLeeway               = 30 * time.Second
)

type RealmAccess struct {
	Roles []string `json:"roles"`
}

type KeycloakClaims struct {
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

func (u *UserContext) HasRole(roles ...string) bool {
	for _, role := range roles {
		if slices.Contains(u.Roles, role) {
			return true
		}
	}
	return false
}

type KeyProvider interface {
	Key(ctx context.Context, kid string) (*rsa.PublicKey, error)
}

type Verifier struct {
	keys   KeyProvider
	parser *jwt.Parser
}

func NewVerifier(issuer string, keys KeyProvider) *Verifier {
	return &Verifier{
		keys: keys,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(issuer),
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(clockLeeway),
		),
	}
}

func KeycloakIssuer(baseURL, realm string) string {
	return strings.TrimRight(baseURL, "/") + "/realms/" + realm
}

func (v *Verifier) Verify(ctx context.Context, tokenString string) (*UserContext, error) {
	var claims KeycloakClaims
	_, err := v.parser.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errUnknownKey
		}
		return v.keys.Key(ctx, kid)
	})
	if err != nil {
		return nil, err
	}
	if claims.Subject == "" {
		return nil, errors.New("token without subject")
	}
	return &UserContext{
		ID:       claims.Subject,
		Username: claims.PreferredUsername,
		Email:    claims.Email,
		Roles:    append([]string{}, claims.RealmAccess.Roles...),
	}, nil
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Authenticate(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := bearerToken(r)
			if tokenString == "" {
				next.ServeHTTP(w, r)
				return
			}
			user, err := v.Verify(r.Context(), tokenString)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid_token", "Token inválido o expirado")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserContextKey, user)))
		})
	}
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUserContext(r.Context()) == nil {
			writeAuthError(w, http.StatusUnauthorized, "unauthorized", "Se requiere autenticación")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !GetUserContext(r.Context()).HasRole(roles...) {
				writeAuthError(w, http.StatusForbidden, "forbidden", "No tienes permiso para esta acción")
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

func bearerToken(r *http.Request) string {
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(token)
	}
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return r.URL.Query().Get("access_token")
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"success":false,"data":null,"error":{"code":"` + code + `","message":"` + message + `"},"meta":null}`))
}

func GetUserContext(ctx context.Context) *UserContext {
	if uc, ok := ctx.Value(UserContextKey).(*UserContext); ok {
		return uc
	}
	return nil
}
