package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNormalizeRoute(t *testing.T) {
	tests := map[string]string{
		"/bookings/booking-1":            "/bookings/{id}",
		"/bookings/booking-1/cancel":     "/bookings/{id}/cancel",
		"/bookings/booking-1/confirm":    "/bookings/{id}/confirm",
		"/payments/payment-1":            "/payments/{id}",
		"/payments/payment-1/process":    "/payments/{id}/process",
		"/payments/payment-1/refund":     "/payments/{id}/refund",
		"/payments/user/user-1":          "/payments/user/{id}",
		"/payments/booking/booking-1":    "/payments/booking/{id}",
		"/availability/doctors/doctor-1": "/availability/doctors/{id}",
		"/auth/users/user-1":             "/auth/users/{id}",
		"/auth/login":                    "/auth/login",
		"/desconocida":                   "not_found",
	}

	for path, want := range tests {
		if got := normalizeRoute(path); got != want {
			t.Fatalf("normalizeRoute(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestMetricsMiddlewareRecordsDefaultStatus200(t *testing.T) {
	before := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/auth/login", "200"))

	handler := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	after := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/auth/login", "200"))
	if after != before+1 {
		t.Fatalf("expected counter to increment by 1, before=%v after=%v", before, after)
	}
}
