package orders

import (
	"bytes"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCalculateDistanceKM(t *testing.T) {
	lat1, lon1 := 10.4910, -66.8620
	lat2, lon2 := 10.4930, -66.8550
	dist := CalculateDistanceKM(lat1, lon1, lat2, lon2)
	if dist <= 0 || dist > 2.0 {
		t.Fatalf("unexpected distance in Las Mercedes: %f km", dist)
	}
}

func TestCalculateDeliveryFeeMinMax(t *testing.T) {
	feeZero := CalculateDeliveryFee(0)
	if feeZero != 1.50 {
		t.Fatalf("expected minimum fee 1.50, got %f", feeZero)
	}

	feeShort := CalculateDeliveryFee(2.0)
	if math.Abs(feeShort-2.00) > 0.01 {
		t.Fatalf("expected fee 2.00 for 2km, got %f", feeShort)
	}

	feeLong := CalculateDeliveryFee(25.0)
	if feeLong != 8.00 {
		t.Fatalf("expected maximum fee 8.00, got %f", feeLong)
	}
}

func TestEstimateDeliveryValidation(t *testing.T) {
	h := &Handler{repo: nil}

	reqNoStore := httptest.NewRequest(http.MethodGet, "/estimate?lat=10.49&lng=-66.85", nil)
	wNoStore := httptest.NewRecorder()
	h.EstimateDelivery(wNoStore, reqNoStore)
	if wNoStore.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing store_id, got %d", wNoStore.Code)
	}

	reqInvalidCoords := httptest.NewRequest(http.MethodGet, "/estimate?store-id=00000000-0000-0000-0000-000000000001&lat=abc&lng=def", nil)
	wInvalidCoords := httptest.NewRecorder()
	h.EstimateDelivery(wInvalidCoords, reqInvalidCoords)
	if wInvalidCoords.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid coordinates, got %d", wInvalidCoords.Code)
	}
}

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

func TestExecuteActionEmptyAction(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodPost, "/00000000-0000-0000-0000-000000000001/actions", bytes.NewBufferString(`{"action": ""}`))
	w := httptest.NewRecorder()

	h.ExecuteAction(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
