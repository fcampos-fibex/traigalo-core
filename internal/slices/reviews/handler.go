package reviews

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	r.Get("/", h.List)
	r.Post("/", h.Create)
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

	var storeID, driverID *uuid.UUID
	storeIDParam := r.URL.Query().Get("store-id")
	if storeIDParam == "" {
		storeIDParam = r.URL.Query().Get("store_id")
	}
	if sid, err := uuid.Parse(storeIDParam); err == nil {
		storeID = &sid
	}

	driverIDParam := r.URL.Query().Get("driver-id")
	if driverIDParam == "" {
		driverIDParam = r.URL.Query().Get("driver_id")
	}
	if did, err := uuid.Parse(driverIDParam); err == nil {
		driverID = &did
	}

	reviewsList, err := h.repo.List(r.Context(), storeID, driverID, size, offset)
	if err != nil {
		http.Error(w, `{"error": "failed to list reviews"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reviewsList)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.OrderID == uuid.Nil || req.StoreID == uuid.Nil || req.Rating < 1 || req.Rating > 5 {
		http.Error(w, `{"error": "order_id, store_id and rating between 1 and 5 are required"}`, http.StatusBadRequest)
		return
	}

	if req.CustomerID == nil {
		userCtx := middleware.GetUserContext(r.Context())
		if userCtx != nil && userCtx.ID != "" {
			if cid, err := uuid.Parse(userCtx.ID); err == nil {
				req.CustomerID = &cid
			}
		}
	}

	review, err := h.repo.Create(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "failed to create review: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(review)
}
