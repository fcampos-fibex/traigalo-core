package drivers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindNearbyValidation(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodGet, "/nearby?latitude=invalid&longitude=10.0", nil)
	w := httptest.NewRecorder()

	h.FindNearby(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateLocationInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.UpdateLocation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
