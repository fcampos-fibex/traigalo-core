package tracking

import (
	"encoding/json"
	"net/http"

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
	r.Get("/{orderId}/route", h.GetOrderRoute)
	return r
}

func (h *Handler) GetOrderRoute(w http.ResponseWriter, r *http.Request) {
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid order id"}`, http.StatusBadRequest)
		return
	}

	route, err := h.repo.GetOrderRoute(r.Context(), orderID)
	if err != nil {
		http.Error(w, `{"error": "failed to calculate order route: `+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(route)
}
