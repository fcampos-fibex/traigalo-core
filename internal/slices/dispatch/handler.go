package dispatch

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
	r.Post("/assign", h.Assign)
	r.Post("/offers/{offerId}/actions", h.ExecuteAction)
	return r
}

func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	var req AssignDispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.OrderID == uuid.Nil {
		http.Error(w, `{"error": "order_id is required"}`, http.StatusBadRequest)
		return
	}

	offer, err := h.repo.AssignNearest(r.Context(), req.OrderID)
	if err != nil {
		http.Error(w, `{"error": "failed to assign dispatch: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(offer)
}

func (h *Handler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	offerIDStr := chi.URLParam(r, "offerId")
	offerID, err := uuid.Parse(offerIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid offer id"}`, http.StatusBadRequest)
		return
	}

	var req OfferActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.Action != "accept" && req.Action != "reject" {
		http.Error(w, `{"error": "action must be 'accept' or 'reject'"}`, http.StatusBadRequest)
		return
	}

	res, err := h.repo.ExecuteOfferAction(r.Context(), offerID, req)
	if err != nil {
		http.Error(w, `{"error": "failed to execute offer action: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"offer_id": offerID,
		"action":   req.Action,
		"status":   "executed",
		"result":   res,
	})
}
