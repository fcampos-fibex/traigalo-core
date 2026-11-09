package dispatch

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssignInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/assign", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.Assign(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAssignMissingOrderID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/assign", bytes.NewBufferString(`{"order_id": "00000000-0000-0000-0000-000000000000"}`))
	w := httptest.NewRecorder()

	h.Assign(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestExecuteActionInvalidUUID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/offers/not-a-uuid/actions", bytes.NewBufferString(`{"action": "accept"}`))
	w := httptest.NewRecorder()

	h.ExecuteAction(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestExecuteActionInvalidAction(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/offers/00000000-0000-0000-0000-000000000001/actions", bytes.NewBufferString(`{"action": "unknown"}`))
	w := httptest.NewRecorder()

	h.ExecuteAction(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
