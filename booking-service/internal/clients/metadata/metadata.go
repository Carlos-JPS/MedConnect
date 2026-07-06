package metadata

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const requestIDKey = "x-request-id"

func ContextWithRequestID(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}

	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(requestIDKey); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if values := md.Get(requestIDKey); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}

	return ""
}
