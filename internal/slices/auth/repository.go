package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/traigalo/traigalo-core/internal/platform/middleware"
)

type Repository struct {
	pool       *pgxpool.Pool
	httpClient *http.Client
}

type keycloakTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func keycloakBaseURL() string {
	if v := os.Getenv("KEYCLOAK_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://auth.traigalo.org"
}

func keycloakRealm() string {
	if v := os.Getenv("KEYCLOAK_REALM"); v != "" {
		return v
	}
	return "traigalo"
}

func keycloakClientID() string {
	if v := os.Getenv("KEYCLOAK_CLIENT_ID"); v != "" {
		return v
	}
	return "traigalo-web"
}

func (r *Repository) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	if firstName == "" && req.FullName != "" {
		parts := strings.Fields(req.FullName)
		if len(parts) > 0 {
			firstName = parts[0]
		}
		if len(parts) > 1 {
			lastName = strings.Join(parts[1:], " ")
		} else {
			lastName = "Traigalo"
		}
	}
	if firstName == "" {
		firstName = "Usuario"
	}
	if lastName == "" {
		lastName = "Traigalo"
	}

	kcUserID, err := r.provisionKeycloakUser(ctx, req.Email, req.Password, firstName, lastName, req.Role)
	if err != nil {
		return nil, fmt.Errorf("keycloak provisioning error: %w", err)
	}

	tokResp, err := r.exchangePasswordGrant(ctx, req.Email, req.Password)
	if err != nil {
		return nil, fmt.Errorf("keycloak token error: %w", err)
	}

	user, err := r.syncUserInDB(ctx, kcUserID, req.Email, req.Phone, firstName, lastName, req.Role)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token:        tokResp.AccessToken,
		RefreshToken: tokResp.RefreshToken,
		AuthProvider: "keycloak_oidc",
		Issuer:       fmt.Sprintf("%s/realms/%s", keycloakBaseURL(), keycloakRealm()),
		User:         *user,
	}, nil
}

func (r *Repository) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	var tokResp *keycloakTokenResponse
	var err error

	if req.Code != "" {
		tokResp, err = r.exchangeAuthorizationCode(ctx, req.Code, req.RedirectURI)
	} else {
		tokResp, err = r.exchangePasswordGrant(ctx, req.Email, req.Password)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid keycloak credentials: %w", err)
	}

	parser := jwt.NewParser()
	var claims middleware.KeycloakClaims
	_, _, _ = parser.ParseUnverified(tokResp.AccessToken, &claims)

	email := claims.Email
	if email == "" {
		email = claims.PreferredUsername
	}
	if email == "" {
		email = req.Email
	}

	kcID := claims.Sub
	if kcID == "" {
		kcID = "kc_" + uuid.New().String()
	}

	role := "customer"
	for _, rl := range claims.RealmAccess.Roles {
		if rl == "admin" || rl == "vendor" || rl == "driver" || rl == "customer" {
			role = rl
			break
		}
	}

	firstName := claims.GivenName
	lastName := claims.FamilyName
	if firstName == "" {
		firstName = strings.Split(email, "@")[0]
	}
	if lastName == "" {
		lastName = "Traigalo"
	}

	user, err := r.syncUserInDB(ctx, kcID, email, "", firstName, lastName, role)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token:        tokResp.AccessToken,
		RefreshToken: tokResp.RefreshToken,
		AuthProvider: "keycloak_oidc",
		Issuer:       fmt.Sprintf("%s/realms/%s", keycloakBaseURL(), keycloakRealm()),
		User:         *user,
	}, nil
}

func (r *Repository) exchangePasswordGrant(ctx context.Context, username, password string) (*keycloakTokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", keycloakBaseURL(), keycloakRealm())
	form := url.Values{}
	form.Set("client_id", keycloakClientID())
	form.Set("grant_type", "password")
	form.Set("username", username)
	form.Set("password", password)
	form.Set("scope", "openid profile email")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tok keycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 || tok.AccessToken == "" {
		return nil, fmt.Errorf("%s: %s", tok.Error, tok.ErrorDesc)
	}
	return &tok, nil
}

func (r *Repository) exchangeAuthorizationCode(ctx context.Context, code, redirectURI string) (*keycloakTokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", keycloakBaseURL(), keycloakRealm())
	if redirectURI == "" {
		redirectURI = "https://traigalo.org/login"
	}
	form := url.Values{}
	form.Set("client_id", keycloakClientID())
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tok keycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 || tok.AccessToken == "" {
		return nil, fmt.Errorf("%s: %s", tok.Error, tok.ErrorDesc)
	}
	return &tok, nil
}

func (r *Repository) getMasterAdminToken(ctx context.Context) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", keycloakBaseURL())
	adminUser := os.Getenv("KEYCLOAK_ADMIN")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("KEYCLOAK_ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "AdminKeycloak2026!"
	}

	form := url.Values{}
	form.Set("client_id", "admin-cli")
	form.Set("grant_type", "password")
	form.Set("username", adminUser)
	form.Set("password", adminPass)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tok keycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("failed to obtain keycloak admin token")
	}
	return tok.AccessToken, nil
}

func (r *Repository) provisionKeycloakUser(ctx context.Context, email, password, firstName, lastName, role string) (string, error) {
	adminToken, err := r.getMasterAdminToken(ctx)
	if err != nil {
		return "", err
	}

	usersURL := fmt.Sprintf("%s/admin/realms/%s/users", keycloakBaseURL(), keycloakRealm())
	payload := map[string]any{
		"username":      email,
		"email":         email,
		"firstName":     firstName,
		"lastName":      lastName,
		"enabled":       true,
		"emailVerified": true,
		"credentials": []map[string]any{
			{
				"type":      "password",
				"value":     password,
				"temporary": false,
			},
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost, usersURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+adminToken)

	createResp, err := r.httpClient.Do(createReq)
	if err != nil {
		return "", err
	}
	_, _ = io.Copy(io.Discard, createResp.Body)
	createResp.Body.Close()

	searchURL := fmt.Sprintf("%s/admin/realms/%s/users?email=%s", keycloakBaseURL(), keycloakRealm(), url.QueryEscape(email))
	searchReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	searchReq.Header.Set("Authorization", "Bearer "+adminToken)
	searchResp, err := r.httpClient.Do(searchReq)
	if err != nil {
		return "", err
	}
	defer searchResp.Body.Close()

	var found []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(searchResp.Body).Decode(&found); err != nil || len(found) == 0 {
		return "", fmt.Errorf("could not locate user in keycloak after creation")
	}
	kcUserID := found[0].ID

	roleURL := fmt.Sprintf("%s/admin/realms/%s/roles/%s", keycloakBaseURL(), keycloakRealm(), role)
	roleReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, roleURL, nil)
	roleReq.Header.Set("Authorization", "Bearer "+adminToken)
	roleResp, err := r.httpClient.Do(roleReq)
	if err == nil {
		var roleObj map[string]any
		if json.NewDecoder(roleResp.Body).Decode(&roleObj) == nil && roleObj["id"] != nil {
			mapURL := fmt.Sprintf("%s/admin/realms/%s/users/%s/role-mappings/realm", keycloakBaseURL(), keycloakRealm(), kcUserID)
			mapBytes, _ := json.Marshal([]map[string]any{{"id": roleObj["id"], "name": roleObj["name"]}})
			mapReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, mapURL, bytes.NewReader(mapBytes))
			mapReq.Header.Set("Content-Type", "application/json")
			mapReq.Header.Set("Authorization", "Bearer "+adminToken)
			mapRes, err := r.httpClient.Do(mapReq)
			if err == nil {
				mapRes.Body.Close()
			}
		}
		roleResp.Body.Close()
	}

	return kcUserID, nil
}

func (r *Repository) syncUserInDB(ctx context.Context, kcID, email, phone, firstName, lastName, role string) (*User, error) {
	query := `
		INSERT INTO users (keycloak_id, email, phone, first_name, last_name, role, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		ON CONFLICT (email) DO UPDATE
		SET keycloak_id = EXCLUDED.keycloak_id,
		    first_name = CASE WHEN EXCLUDED.first_name <> '' THEN EXCLUDED.first_name ELSE users.first_name END,
		    last_name = CASE WHEN EXCLUDED.last_name <> '' THEN EXCLUDED.last_name ELSE users.last_name END,
		    role = EXCLUDED.role,
		    updated_at = NOW()
		RETURNING id, keycloak_id, email, COALESCE(phone, ''), first_name, last_name, role, is_active, created_at
	`
	var u User
	err := r.pool.QueryRow(ctx, query, kcID, email, phone, firstName, lastName, role).Scan(
		&u.ID, &u.KeycloakID, &u.Email, &u.Phone, &u.FirstName, &u.LastName, &u.Role, &u.IsActive, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.FullName = strings.TrimSpace(u.FirstName + " " + u.LastName)
	return &u, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, keycloak_id, email, COALESCE(phone, ''), first_name, last_name, role, is_active, created_at
		FROM users
		WHERE id = $1
	`
	var user User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.KeycloakID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
		&user.Role, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	user.FullName = strings.TrimSpace(user.FirstName + " " + user.LastName)
	return &user, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, keycloak_id, email, COALESCE(phone, ''), first_name, last_name, role, is_active, created_at
		FROM users
		WHERE email = $1
	`
	var user User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.KeycloakID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
		&user.Role, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	user.FullName = strings.TrimSpace(user.FirstName + " " + user.LastName)
	return &user, nil
}
