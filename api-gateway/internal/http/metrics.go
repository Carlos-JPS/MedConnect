package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "medconnect_api_gateway_http_requests_total",
			Help: "Total de requests HTTP procesadas por el API Gateway.",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "medconnect_api_gateway_http_request_duration_seconds",
			Help:    "Duración de requests HTTP procesadas por el API Gateway.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusResponseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}

		route := normalizeRoute(r.URL.Path)
		labels := []string{r.Method, route, strconv.Itoa(status)}
		httpRequestsTotal.WithLabelValues(labels...).Inc()
		httpRequestDuration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func normalizeRoute(path string) string {
	switch {
	case path == "/auth/register", path == "/auth/login", path == "/auth/validate", path == "/bookings", path == "/payments", path == "/availability/slots":
		return path
	case isSingleSegmentPath(path, "/auth/users/"):
		return "/auth/users/{id}"
	case isBookingRoute(path, "/cancel"):
		return "/bookings/{id}/cancel"
	case isBookingRoute(path, "/confirm"):
		return "/bookings/{id}/confirm"
	case isSingleSegmentPath(path, "/bookings/"):
		return "/bookings/{id}"
	case isPaymentRoute(path, "/process"):
		return "/payments/{id}/process"
	case isPaymentRoute(path, "/refund"):
		return "/payments/{id}/refund"
	case isSingleSegmentPath(path, "/payments/user/"):
		return "/payments/user/{id}"
	case isSingleSegmentPath(path, "/payments/booking/"):
		return "/payments/booking/{id}"
	case isSingleSegmentPath(path, "/payments/"):
		return "/payments/{id}"
	case isSingleSegmentPath(path, "/availability/doctors/"):
		return "/availability/doctors/{id}"
	default:
		return "not_found"
	}
}

func isBookingRoute(path, suffix string) bool {
	return hasSingleIDBetween(path, "/bookings/", suffix)
}

func isPaymentRoute(path, suffix string) bool {
	return hasSingleIDBetween(path, "/payments/", suffix)
}

func isSingleSegmentPath(path, prefix string) bool {
	trimmed := strings.TrimPrefix(path, prefix)
	return trimmed != path && trimmed != "" && !strings.Contains(trimmed, "/")
}

func hasSingleIDBetween(path, prefix, suffix string) bool {
	trimmed := strings.TrimPrefix(path, prefix)
	if trimmed == path || trimmed == "" || !strings.HasSuffix(trimmed, suffix) {
		return false
	}
	trimmed = strings.TrimSuffix(trimmed, suffix)
	return trimmed != "" && !strings.Contains(trimmed, "/")
}
