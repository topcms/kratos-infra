package redis

import (
	"context"
	"time"

	infralogging "github.com/topcms/kratos-infra/middleware/logging"

	"github.com/go-kratos/kratos/v2/log"
	goredis "github.com/redis/go-redis/v9"
)

type loggingHook struct {
	helper *log.Helper
}

func newLoggingHook(base log.Logger) goredis.Hook {
	return &loggingHook{
		helper: log.NewHelper(log.With(base, "module", "redis", "db.system", "redis")),
	}
}

func (h *loggingHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return next
}

func (h *loggingHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		begin := time.Now()
		err := next(ctx, cmd)
		traceID := infralogging.TraceIDFromContext(ctx)
		latency := time.Since(begin)

		fields := []interface{}{
			"trace_id", traceID,
			"latency", latency,
			"cmd", cmd.Name(),
			"args", cmd.String(),
		}
		if err != nil {
			h.helper.Errorw(append(fields, "error", err)...)
			return err
		}
		h.helper.Infow(fields...)
		return nil
	}
}

func (h *loggingHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		begin := time.Now()
		err := next(ctx, cmds)
		traceID := infralogging.TraceIDFromContext(ctx)
		latency := time.Since(begin)

		fields := []interface{}{
			"trace_id", traceID,
			"latency", latency,
			"cmd_count", len(cmds),
		}
		if err != nil {
			h.helper.Errorw(append(fields, "error", err)...)
			return err
		}
		h.helper.Infow(fields...)
		return nil
	}
}
