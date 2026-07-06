package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

var requestIDKey = requestIDContextKey{}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}

		w.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(&requestIDResponseWriter{ResponseWriter: w, requestID: requestID}, r.WithContext(ctx))
	})
}

type requestIDResponseWriter struct {
	http.ResponseWriter
	requestID string
}

func (w *requestIDResponseWriter) ensureHeader() {
	w.Header().Set(requestIDHeader, w.requestID)
}

func (w *requestIDResponseWriter) WriteHeader(status int) {
	w.ensureHeader()
	w.ResponseWriter.WriteHeader(status)
}

func (w *requestIDResponseWriter) Write(p []byte) (int, error) {
	w.ensureHeader()
	return w.ResponseWriter.Write(p)
}

func generateRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
