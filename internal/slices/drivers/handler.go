package drivers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
	hub  *Hub
}

func NewHandler(repo *Repository, hub *Hub) *Handler {
	return &Handler{
		repo: repo,
		hub:  hub,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.UpdateLocation)
	r.Get("/nearby", h.FindNearby)
	r.Get("/telemetry", h.TelemetryWebSocket)
	return r
}

func (h *Handler) TelemetryWebSocket(w http.ResponseWriter, r *http.Request) {
	if h.hub != nil {
		h.hub.HandleWebSocket(w, r)
	}
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var update LocationUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, `{"error": "invalid request payload"}`, http.StatusBadRequest)
		return
	}

	update.Timestamp = time.Now().UTC()

	if err := h.repo.UpdateLocation(r.Context(), update); err != nil {
		http.Error(w, `{"error": "failed to update driver location"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

func (h *Handler) FindNearby(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("latitude")
	lngStr := r.URL.Query().Get("longitude")
	radiusStr := r.URL.Query().Get("radius-km")

	lat, err1 := strconv.ParseFloat(latStr, 64)
	lng, err2 := strconv.ParseFloat(lngStr, 64)
	if err1 != nil || err2 != nil {
		http.Error(w, `{"error": "latitude and longitude are required numbers"}`, http.StatusBadRequest)
		return
	}

	radiusKM, err3 := strconv.ParseFloat(radiusStr, 64)
	if err3 != nil || radiusKM <= 0 {
		radiusKM = 5.0
	}

	count := 20
	driversList, err := h.repo.FindNearby(r.Context(), lat, lng, radiusKM, count)
	if err != nil {
		http.Error(w, `{"error": "failed to find nearby drivers"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(driversList)
}
