package grpcmeta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/MedConnect/api-gateway/internal/http"
	"google.golang.org/grpc/metadata"
)

func TestContextWithRequestIDAddsOutgoingMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "request-123")

	var got context.Context
	handler := httpapi.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = ContextWithRequestID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	md, ok := metadata.FromOutgoingContext(got)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	vals := md.Get(requestIDKey)
	if len(vals) != 1 || vals[0] != "request-123" {
		t.Fatalf("expected request id metadata, got %v", vals)
	}
}
