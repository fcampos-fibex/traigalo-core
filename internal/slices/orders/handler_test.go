package orders

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateOrderInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateOrderEmptyItems(t *testing.T) {
	h := &Handler{repo: nil}

	payload := `{"customer_id": "00000000-0000-0000-0000-000000000001", "store_id": "00000000-0000-0000-0000-000000000002", "items": []}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
