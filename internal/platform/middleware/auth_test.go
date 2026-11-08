package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGatewayAuthDisabled(t *testing.T) {
	_ = os.Setenv("AUTH_ENABLED", "false")
	defer os.Unsetenv("AUTH_ENABLED")

	var capturedUser *UserContext
	handler := GatewayAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = GetUserContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if capturedUser == nil {
		t.Fatal("expected user in context when AUTH_ENABLED=false")
	}
	if capturedUser.ID != "dev-bypass-user" {
		t.Fatalf("expected user dev-bypass-user, got %s", capturedUser.ID)
	}
}

func TestGatewayAuthKeycloakJWT(t *testing.T) {
	claims := KeycloakClaims{
		Sub:               "keycloak-user-uuid-1234",
		PreferredUsername: "alexander_backend",
		Email:             "alexander@traigalo.org",
		RealmAccess: RealmAccess{
			Roles: []string{"admin", "vendor"},
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("dev-secret-key-test"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_ = os.Setenv("JWT_SECRET", "dev-secret-key-test")
	defer os.Unsetenv("JWT_SECRET")

	var capturedUser *UserContext
	handler := GatewayAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = GetUserContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if capturedUser == nil {
		t.Fatal("expected user context from Keycloak JWT")
	}
	if capturedUser.ID != "keycloak-user-uuid-1234" {
		t.Fatalf("expected sub keycloak-user-uuid-1234, got %s", capturedUser.ID)
	}
	if capturedUser.Username != "alexander_backend" {
		t.Fatalf("expected username alexander_backend, got %s", capturedUser.Username)
	}
	if capturedUser.Email != "alexander@traigalo.org" {
		t.Fatalf("expected email alexander@traigalo.org, got %s", capturedUser.Email)
	}
	if len(capturedUser.Roles) != 2 || capturedUser.Roles[0] != "admin" {
		t.Fatalf("unexpected roles: %v", capturedUser.Roles)
	}
}
