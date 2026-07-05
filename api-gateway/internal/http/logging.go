package httpapi

import (
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusResponseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}

		log.Printf(
			"request_id=%s method=%s route=%s status=%d duration=%s",
			RequestIDFromContext(r.Context()),
			r.Method,
			r.URL.Path,
			status,
			time.Since(start),
		)
	})
}
