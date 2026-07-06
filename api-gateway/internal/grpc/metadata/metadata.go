package grpcmeta

import (
	"context"

	httpapi "github.com/MedConnect/api-gateway/internal/http"
	"google.golang.org/grpc/metadata"
)

const requestIDKey = "x-request-id"

func ContextWithRequestID(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}

	requestID := httpapi.RequestIDFromContext(ctx)
	if requestID == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, requestIDKey, requestID)
}
