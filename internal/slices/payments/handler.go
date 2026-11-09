package payments

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
	r.Post("/receipts", h.CreateReceipt)
	r.Get("/receipts", h.ListReceipts)
	r.Post("/receipts/{receiptId}/actions", h.ExecuteAction)
	return r
}

func (h *Handler) CreateReceipt(w http.ResponseWriter, r *http.Request) {
	var req CreateReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.OrderID == uuid.Nil || req.PaymentMethod == "" || req.ReferenceNumber == "" {
		http.Error(w, `{"error": "order_id, payment_method and reference_number are required"}`, http.StatusBadRequest)
		return
	}

	receipt, err := h.repo.CreateReceipt(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "failed to create payment receipt: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(receipt)
}

func (h *Handler) ListReceipts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var orderID *uuid.UUID
	orderIDParam := r.URL.Query().Get("order-id")
	if orderIDParam == "" {
		orderIDParam = r.URL.Query().Get("order_id")
	}
	if oid, err := uuid.Parse(orderIDParam); err == nil {
		orderID = &oid
	}

	receipts, err := h.repo.ListReceipts(r.Context(), orderID, size, offset)
	if err != nil {
		http.Error(w, `{"error": "failed to list receipts"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(receipts)
}

func (h *Handler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	receiptIDStr := chi.URLParam(r, "receiptId")
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid receipt id"}`, http.StatusBadRequest)
		return
	}

	var req ReceiptActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.Action != "verify" && req.Action != "reject" {
		http.Error(w, `{"error": "action must be 'verify' or 'reject'"}`, http.StatusBadRequest)
		return
	}

	err = h.repo.ExecuteReceiptAction(r.Context(), receiptID, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to execute receipt action: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"receipt_id": receiptID,
		"action":     req.Action,
		"status":     "executed",
	})
}
