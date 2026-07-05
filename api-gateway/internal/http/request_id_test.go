package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddlewarePreservesIncomingHeader(t *testing.T) {
	const expected = "request-123"

	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != expected {
			t.Fatalf("expected request id in context %q, got %q", expected, got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestIDHeader, expected)

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(requestIDHeader); got != expected {
		t.Fatalf("expected response header %q, got %q", expected, got)
	}
}

func TestRequestIDMiddlewareGeneratesRequestID(t *testing.T) {
	var contextRequestID string

	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextRequestID = RequestIDFromContext(r.Context())
		if contextRequestID == "" {
			t.Fatal("expected generated request id in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(requestIDHeader); got == "" {
		t.Fatal("expected response header X-Request-ID")
	} else if got != contextRequestID {
		t.Fatalf("expected response header to match context request id %q, got %q", contextRequestID, got)
	}
}

func TestRequestIDFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, "context-request-id")
	if got := RequestIDFromContext(ctx); got != "context-request-id" {
		t.Fatalf("expected request id from context, got %q", got)
	}
}
