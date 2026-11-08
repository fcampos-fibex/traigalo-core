package orders

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
	r.Get("/{orderId}", h.GetByID)
	r.Post("/{orderId}/actions", h.ExecuteAction)
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

	status := r.URL.Query().Get("status")

	var customerID, storeID, driverID *uuid.UUID
	if cid, err := uuid.Parse(r.URL.Query().Get("customer-id")); err == nil {
		customerID = &cid
	}
	if sid, err := uuid.Parse(r.URL.Query().Get("store-id")); err == nil {
		storeID = &sid
	}
	if did, err := uuid.Parse(r.URL.Query().Get("driver-id")); err == nil {
		driverID = &did
	}

	ordersList, err := h.repo.List(r.Context(), status, customerID, storeID, driverID, size, offset)
	if err != nil {
		http.Error(w, `{"error": "failed to list orders"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ordersList)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid order id"}`, http.StatusBadRequest)
		return
	}

	order, err := h.repo.GetByID(r.Context(), orderID)
	if err != nil {
		http.Error(w, `{"error": "order not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(order)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, `{"error": "order must contain at least one item"}`, http.StatusBadRequest)
		return
	}

	order, err := h.repo.Create(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "failed to create order"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(order)
}

func (h *Handler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid order id"}`, http.StatusBadRequest)
		return
	}

	var req OrderActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.Action == "" {
		http.Error(w, `{"error": "action is required"}`, http.StatusBadRequest)
		return
	}

	err = h.repo.ExecuteAction(r.Context(), orderID, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error": "invalid state transition or order not found"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error": "failed to execute action: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"order_id": orderID,
		"action":   req.Action,
		"status":   "executed",
	})
}
