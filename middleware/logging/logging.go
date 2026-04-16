package logging

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/google/uuid"
)

func Server(logger log.Logger) middleware.Middleware {
	helper := log.NewHelper(log.With(logger, "module", "middleware/logging"))

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			traceID := TraceIDFromContext(ctx)
			if traceID == "" {
				traceID = uuid.NewString()
			}

			begin := time.Now()
			reply, err := handler(ctx, req)
			cost := time.Since(begin)

			if err != nil {
				helper.Errorw("trace_id", traceID, "latency", cost, "error", err)
				return nil, err
			}

			helper.Infow("trace_id", traceID, "latency", cost)
			return reply, nil
		}
	}
}

type traceIDKey struct{}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

func TraceIDFromContext(ctx context.Context) string {
	value := ctx.Value(traceIDKey{})
	if value == nil {
		return ""
	}

	id, ok := value.(string)
	if !ok {
		return ""
	}
	return id
}
