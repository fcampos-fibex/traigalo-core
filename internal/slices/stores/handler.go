package stores

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
	r.Get("/{storeId}", h.GetByID)
	r.Get("/{storeId}/products", h.ListProducts)
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

	onlyOpen := r.URL.Query().Get("status") == "open"

	storesList, err := h.repo.List(r.Context(), onlyOpen, size, offset)
	if err != nil {
		http.Error(w, `{"error": "failed to list stores"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(storesList)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	storeIDStr := chi.URLParam(r, "storeId")
	storeID, err := uuid.Parse(storeIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid store id"}`, http.StatusBadRequest)
		return
	}

	store, err := h.repo.GetByID(r.Context(), storeID)
	if err != nil {
		http.Error(w, `{"error": "store not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(store)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	storeIDStr := chi.URLParam(r, "storeId")
	storeID, err := uuid.Parse(storeIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid store id"}`, http.StatusBadRequest)
		return
	}

	productsList, err := h.repo.ListProducts(r.Context(), storeID)
	if err != nil {
		http.Error(w, `{"error": "failed to list products"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(productsList)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateStoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Slug == "" || req.AddressLine == "" {
		http.Error(w, `{"error": "missing required store fields"}`, http.StatusBadRequest)
		return
	}

	store, err := h.repo.Create(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "failed to create store"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(store)
}
