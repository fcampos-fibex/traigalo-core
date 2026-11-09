package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/traigalo/traigalo-core/internal/platform/middleware"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/login", h.Login)
	r.Post("/register", h.Register)
	r.Get("/me", h.Me)
	return r
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" {
		http.Error(w, `{"error": "email, password, first_name and last_name are required"}`, http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = "customer"
	}

	if req.Role != "customer" && req.Role != "vendor" && req.Role != "driver" && req.Role != "admin" {
		http.Error(w, `{"error": "invalid role"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.repo.Register(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "failed to register user: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, `{"error": "email is required"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.repo.Login(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserContext(r.Context())
	if userCtx == nil || userCtx.ID == "" {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if parsedID, err := uuid.Parse(userCtx.ID); err == nil {
		user, err := h.repo.GetByID(r.Context(), parsedID)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(user)
			return
		}
	}

	if userCtx.Email != "" {
		user, err := h.repo.GetByEmail(r.Context(), userCtx.Email)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(user)
			return
		}
	}

	role := "customer"
	if len(userCtx.Roles) > 0 {
		role = userCtx.Roles[0]
	}

	profile := map[string]any{
		"id":        userCtx.ID,
		"username":  userCtx.Username,
		"email":     userCtx.Email,
		"roles":     userCtx.Roles,
		"role":      role,
		"is_active": true,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(profile)
}
