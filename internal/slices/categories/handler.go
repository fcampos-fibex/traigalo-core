package categories

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{categoryId}", h.GetByID)
	return r
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	categoriesList, err := h.repo.List(r.Context(), size, offset)
	if err != nil {
		http.Error(w, `{"error": "failed to list categories"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(categoriesList)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := chi.URLParam(r, "categoryId")
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid category id"}`, http.StatusBadRequest)
		return
	}

	category, err := h.repo.GetByID(r.Context(), categoryID)
	if err != nil {
		http.Error(w, `{"error": "category not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(category)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Slug == "" {
		http.Error(w, `{"error": "name and slug are required"}`, http.StatusBadRequest)
		return
	}

	category, err := h.repo.Create(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "failed to create category"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(category)
}
