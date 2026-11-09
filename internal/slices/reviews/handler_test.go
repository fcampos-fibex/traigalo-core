package reviews

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateReviewInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateReviewInvalidRating(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"rating": 10}`))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
