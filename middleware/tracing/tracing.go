package tracing

import (
	"context"

	"kratos-infra/middleware/logging"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
)

const (
	traceIDHeader = "X-Request-Id"
)

func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			traceID := traceIDFromHeader(ctx)
			if traceID == "" {
				traceID = uuid.NewString()
			}

			ctx = logging.WithTraceID(ctx, traceID)
			return handler(ctx, req)
		}
	}
}

func traceIDFromHeader(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	return tr.RequestHeader().Get(traceIDHeader)
}
