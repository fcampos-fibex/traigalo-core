package tracking

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOrderRouteInvalidUUID(t *testing.T) {
	h := &Handler{repo: nil}

	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid/route", nil)
	w := httptest.NewRecorder()

	h.GetOrderRoute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGenerateInterpolatedPolyline(t *testing.T) {
	waypoints := []Waypoint{
		{Type: "store", Latitude: 10.491, Longitude: -66.862},
		{Type: "customer", Latitude: 10.493, Longitude: -66.855},
	}
	polyline := generateInterpolatedPolyline(waypoints)
	if len(polyline) < 2 {
		t.Fatalf("expected polyline points, got %d", len(polyline))
	}
	if polyline[0][0] != 10.491 || polyline[len(polyline)-1][0] != 10.493 {
		t.Fatalf("polyline ends do not match waypoints")
	}
}
