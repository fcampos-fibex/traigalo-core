package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer = "https://auth.example.test/realms/traigalo"
	testKid    = "test-kid"
)

type staticKeys struct{ key *rsa.PublicKey }

func (s staticKeys) Key(_ context.Context, kid string) (*rsa.PublicKey, error) {
	if kid != testKid {
		return nil, errUnknownKey
	}
	return s.key, nil
}

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims KeycloakClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func validClaims() KeycloakClaims {
	return KeycloakClaims{
		PreferredUsername: "rider",
		RealmAccess:       RealmAccess{Roles: []string{"driver"}},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "8d0c7a52-1f7b-4c1e-9d55-2c1a3b4f5e60",
			Issuer:    testIssuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func serve(handler http.Handler, authHeader string) int {
	req := httptest.NewRequest(http.MethodGet, "/v2/orders", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Code
}

func TestProtectedRouteRejectsBadTokens(t *testing.T) {
	key := newTestKey(t)
	otherKey := newTestKey(t)
	protected := Authenticate(NewVerifier(testIssuer, staticKeys{key: &key.PublicKey}))(RequireAuth(okHandler()))

	expired := validClaims()
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
	wrongIssuer := validClaims()
	wrongIssuer.Issuer = "https://evil.example.test/realms/traigalo"
	noExpiry := validClaims()
	noExpiry.ExpiresAt = nil
	hsToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims()).SignedString([]byte("shared"))

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"valid token", "Bearer " + signToken(t, key, testKid, validClaims()), http.StatusOK},
		{"missing token", "", http.StatusUnauthorized},
		{"garbage token", "Bearer not-a-jwt", http.StatusUnauthorized},
		{"expired token", "Bearer " + signToken(t, key, testKid, expired), http.StatusUnauthorized},
		{"wrong issuer", "Bearer " + signToken(t, key, testKid, wrongIssuer), http.StatusUnauthorized},
		{"no expiry", "Bearer " + signToken(t, key, testKid, noExpiry), http.StatusUnauthorized},
		{"foreign signature", "Bearer " + signToken(t, otherKey, testKid, validClaims()), http.StatusUnauthorized},
		{"unknown kid", "Bearer " + signToken(t, key, "other", validClaims()), http.StatusUnauthorized},
		{"hs256 downgrade", "Bearer " + hsToken, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := serve(protected, tc.header); got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestGatewayHeadersAreIgnored(t *testing.T) {
	key := newTestKey(t)
	protected := Authenticate(NewVerifier(testIssuer, staticKeys{key: &key.PublicKey}))(RequireAuth(okHandler()))
	req := httptest.NewRequest(http.MethodGet, "/v2/orders", nil)
	req.Header.Set("X-User-ID", "attacker")
	req.Header.Set("X-User-Roles", "admin")
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestVerifiedUserReachesHandler(t *testing.T) {
	key := newTestKey(t)
	var captured *UserContext
	handler := Authenticate(NewVerifier(testIssuer, staticKeys{key: &key.PublicKey}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = GetUserContext(r.Context())
	}))
	serve(handler, "Bearer "+signToken(t, key, testKid, validClaims()))
	if captured == nil || captured.ID != validClaims().Subject || !captured.HasRole("driver") {
		t.Fatalf("unexpected user context: %+v", captured)
	}
}

func TestRequireRole(t *testing.T) {
	key := newTestKey(t)
	adminOnly := Authenticate(NewVerifier(testIssuer, staticKeys{key: &key.PublicKey}))(RequireRole("admin")(okHandler()))
	admin := validClaims()
	admin.RealmAccess.Roles = []string{"admin"}

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"driver forbidden", "Bearer " + signToken(t, key, testKid, validClaims()), http.StatusForbidden},
		{"admin allowed", "Bearer " + signToken(t, key, testKid, admin), http.StatusOK},
		{"anonymous", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := serve(adminOnly, tc.header); got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestWebSocketTokenFromQuery(t *testing.T) {
	key := newTestKey(t)
	protected := Authenticate(NewVerifier(testIssuer, staticKeys{key: &key.PublicKey}))(RequireAuth(okHandler()))
	req := httptest.NewRequest(http.MethodGet, "/v2/drivers/telemetry?access_token="+signToken(t, key, testKid, validClaims()), nil)
	req.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}
