package categories

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateCategoryInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateCategoryEmptyFields(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name": ""}`))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetByIDInvalidUUID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	w := httptest.NewRecorder()

	h.GetByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
