package products

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateProductInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateProductEmptyFields(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name": ""}`))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetProductByIDInvalidUUID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	w := httptest.NewRecorder()

	h.GetByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateProductInvalidUUID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPut, "/not-a-uuid", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteProductInvalidUUID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodDelete, "/not-a-uuid", nil)
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
