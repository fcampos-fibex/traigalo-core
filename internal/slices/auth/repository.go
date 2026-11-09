package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/traigalo/traigalo-core/internal/platform/middleware"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	keycloakID := "kc_" + uuid.New().String()
	query := `
		INSERT INTO users (keycloak_id, email, phone, first_name, last_name, role, is_active, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6, true, $7)
		RETURNING id, is_active, created_at
	`
	var user User
	user.Email = req.Email
	user.Phone = req.Phone
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Role = req.Role

	err = r.pool.QueryRow(
		ctx, query,
		keycloakID, req.Email, req.Phone, req.FirstName, req.LastName, req.Role, string(hashedPassword),
	).Scan(&user.ID, &user.IsActive, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (r *Repository) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	query := `
		SELECT id, email, phone, first_name, last_name, role, is_active, COALESCE(password_hash, ''), created_at
		FROM users
		WHERE email = $1
	`
	var user User
	var passwordHash string
	err := r.pool.QueryRow(ctx, query, req.Email).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
		&user.Role, &user.IsActive, &passwordHash, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if passwordHash != "" && req.Password != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			return nil, fmt.Errorf("invalid credentials")
		}
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, phone, first_name, last_name, role, is_active, created_at
		FROM users
		WHERE id = $1
	`
	var user User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
		&user.Role, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, phone, first_name, last_name, role, is_active, created_at
		FROM users
		WHERE email = $1
	`
	var user User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
		&user.Role, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func generateToken(user User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "traigalo-jwt-dev-secret-key-2026"
	}

	claims := middleware.KeycloakClaims{
		Sub:               user.ID.String(),
		PreferredUsername: user.Email,
		Email:             user.Email,
		RealmAccess: middleware.RealmAccess{
			Roles: []string{user.Role},
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
