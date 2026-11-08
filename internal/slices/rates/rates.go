package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

const RateCacheKey = "rates:bcv:oficial"
const CacheTTL = 10 * time.Minute

type Service struct {
	redis      *redis.Client
	httpClient *http.Client
}

func NewService(redis *redis.Client) *Service {
	return &Service{
		redis: redis,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *Service) GetOfficialRate(ctx context.Context) (float64, error) {
	if s.redis != nil {
		val, err := s.redis.Get(ctx, RateCacheKey).Float64()
		if err == nil && val > 0 {
			return val, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ve.dolarapi.com/v1/dolares/oficial", nil)
	if err != nil {
		return 896.12, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 896.12, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 896.12, fmt.Errorf("rates API returned status %d", resp.StatusCode)
	}

	var data struct {
		Promedio float64 `json:"promedio"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 896.12, err
	}

	if data.Promedio <= 0 {
		data.Promedio = 896.12
	}

	if s.redis != nil {
		_ = s.redis.Set(ctx, RateCacheKey, data.Promedio, CacheTTL).Err()
	}

	return data.Promedio, nil
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetRate)
	return r
}

func (h *Handler) GetRate(w http.ResponseWriter, r *http.Request) {
	rate, err := h.service.GetOfficialRate(r.Context())
	if err != nil && rate <= 0 {
		http.Error(w, `{"error": "failed to fetch currency rate"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"currency": "USD",
		"target":   "VES",
		"rate":     rate,
		"source":   "BCV",
	})
}
