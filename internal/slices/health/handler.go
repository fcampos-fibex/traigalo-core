package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewHandler(pool *pgxpool.Pool, redis *redis.Client) *Handler {
	return &Handler{pool: pool, redis: redis}
}

type Response struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	services := make(map[string]string)
	status := "healthy"

	if err := h.pool.Ping(ctx); err != nil {
		services["postgres"] = "unhealthy: " + err.Error()
		status = "degraded"
	} else {
		services["postgres"] = "healthy"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "unhealthy: " + err.Error()
		status = "degraded"
	} else {
		services["redis"] = "healthy"
	}

	resp := Response{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Services:  services,
	}

	w.Header().Set("Content-Type", "application/json")
	if status == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(resp)
}
