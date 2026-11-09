package payments

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateReceiptInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/receipts", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.CreateReceipt(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateReceiptMissingFields(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/receipts", bytes.NewBufferString(`{"payment_method": ""}`))
	w := httptest.NewRecorder()

	h.CreateReceipt(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestExecuteActionInvalidUUID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/receipts/not-a-uuid/actions", bytes.NewBufferString(`{"action": "verify"}`))
	w := httptest.NewRecorder()

	h.ExecuteAction(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
