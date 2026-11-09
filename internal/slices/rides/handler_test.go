package rides

import (
	"bytes"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCalculateRideFare(t *testing.T) {
	fareMoto := CalculateRideFare("moto", 5.0)
	if math.Abs(fareMoto-4.00) > 0.01 {
		t.Fatalf("expected moto fare 4.00, got %f", fareMoto)
	}

	fareStandard := CalculateRideFare("standard", 10.0)
	if math.Abs(fareStandard-10.50) > 0.01 {
		t.Fatalf("expected standard fare 10.50, got %f", fareStandard)
	}

	fareComfort := CalculateRideFare("comfort", 10.0)
	if math.Abs(fareComfort-16.00) > 0.01 {
		t.Fatalf("expected comfort fare 16.00, got %f", fareComfort)
	}
}

func TestCreateRideInvalidPayload(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid json}"))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateRideMissingCoords(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"vehicle_type": "standard"}`))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestExecuteActionInvalidUUID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/not-a-uuid/actions", bytes.NewBufferString(`{"action": "start_trip"}`))
	w := httptest.NewRecorder()

	h.ExecuteAction(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
