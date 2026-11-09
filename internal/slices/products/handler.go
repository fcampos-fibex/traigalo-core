package products

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	r.Get("/{productId}", h.GetByID)
	r.Put("/{productId}", h.Update)
	r.Delete("/{productId}", h.Delete)
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

	var storeID, categoryID *uuid.UUID
	storeIDParam := r.URL.Query().Get("store-id")
	if storeIDParam == "" {
		storeIDParam = r.URL.Query().Get("store_id")
	}
	if sid, err := uuid.Parse(storeIDParam); err == nil {
		storeID = &sid
	}

	catIDParam := r.URL.Query().Get("category-id")
	if catIDParam == "" {
		catIDParam = r.URL.Query().Get("category_id")
	}
	if cid, err := uuid.Parse(catIDParam); err == nil {
		categoryID = &cid
	}

	search := r.URL.Query().Get("search")

	productsList, err := h.repo.List(r.Context(), storeID, categoryID, search, size, offset)
	if err != nil {
		http.Error(w, `{"error": "failed to list products"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(productsList)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	productIDStr := chi.URLParam(r, "productId")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid product id"}`, http.StatusBadRequest)
		return
	}

	product, err := h.repo.GetByID(r.Context(), productID)
	if err != nil {
		http.Error(w, `{"error": "product not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(product)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.StoreID == uuid.Nil || req.Name == "" || req.BasePrice < 0 {
		http.Error(w, `{"error": "store_id, name and non-negative base_price are required"}`, http.StatusBadRequest)
		return
	}

	product, err := h.repo.Create(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "failed to create product: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(product)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	productIDStr := chi.URLParam(r, "productId")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid product id"}`, http.StatusBadRequest)
		return
	}

	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	product, err := h.repo.Update(r.Context(), productID, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error": "product not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to update product: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(product)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	productIDStr := chi.URLParam(r, "productId")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid product id"}`, http.StatusBadRequest)
		return
	}

	err = h.repo.Delete(r.Context(), productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error": "product not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to delete product"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
