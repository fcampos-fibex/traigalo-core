package rates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOfficialRateFallback(t *testing.T) {
	service := NewService(nil)
	rate, err := service.GetOfficialRate(context.Background())
	if rate <= 0 {
		t.Fatalf("expected positive rate, got %f, err: %v", rate, err)
	}
}

func TestGetRateHandler(t *testing.T) {
	service := NewService(nil)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.GetRate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
