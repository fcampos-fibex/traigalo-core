package rides

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	r.Post("/", h.Create)
	r.Get("/{rideId}", h.GetByID)
	r.Post("/{rideId}/actions", h.ExecuteAction)
	return r
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.PickupLatitude == 0 || req.PickupLongitude == 0 || req.DropoffLatitude == 0 || req.DropoffLongitude == 0 {
		http.Error(w, `{"error": "pickup and dropoff coordinates are required"}`, http.StatusBadRequest)
		return
	}

	if req.CustomerID == uuid.Nil {
		userCtx := middleware.GetUserContext(r.Context())
		if userCtx != nil && userCtx.ID != "" {
			if cid, err := uuid.Parse(userCtx.ID); err == nil {
				req.CustomerID = cid
			}
		}
	}

	ride, err := h.repo.Create(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "failed to create ride request: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ride)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	rideIDStr := chi.URLParam(r, "rideId")
	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid ride id"}`, http.StatusBadRequest)
		return
	}

	ride, err := h.repo.GetByID(r.Context(), rideID)
	if err != nil {
		http.Error(w, `{"error": "ride not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ride)
}

func (h *Handler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	rideIDStr := chi.URLParam(r, "rideId")
	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid ride id"}`, http.StatusBadRequest)
		return
	}

	var req RideActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.Action == "" {
		http.Error(w, `{"error": "action is required"}`, http.StatusBadRequest)
		return
	}

	if req.Action == "accept" && req.DriverID == nil {
		userCtx := middleware.GetUserContext(r.Context())
		if userCtx != nil && userCtx.ID != "" {
			if did, err := uuid.Parse(userCtx.ID); err == nil {
				req.DriverID = &did
			}
		}
	}

	err = h.repo.ExecuteAction(r.Context(), rideID, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error": "invalid state transition or ride not found"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error": "failed to execute ride action: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ride_id": rideID,
		"action":  req.Action,
		"status":  "executed",
	})
}
