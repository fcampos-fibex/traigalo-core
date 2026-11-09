package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRegisterEmptyFields(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email": ""}`))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLoginInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestMeUnauthorized(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()

	h.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
