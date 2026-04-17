package mysql

import (
	"context"
	"errors"
	"time"

	infralogging "github.com/topcms/kratos-infra/middleware/logging"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type kratosGormLogger struct {
	helper               *log.Helper
	slowThreshold        time.Duration
	ignoreRecordNotFound bool
}

func newGormLogger(base log.Logger, cfg Config) gormlogger.Interface {
	slow := cfg.SlowThreshold
	if slow <= 0 {
		slow = 200 * time.Millisecond
	}
	return &kratosGormLogger{
		helper:               log.NewHelper(log.With(base, "module", "mysql", "db.system", "mysql")),
		slowThreshold:        slow,
		ignoreRecordNotFound: cfg.IgnoreRecordNotFound,
	}
}

func (l *kratosGormLogger) LogMode(_ gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *kratosGormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	traceID := infralogging.TraceIDFromContext(ctx)
	l.helper.Infow("trace_id", traceID, "msg", msg, "args", args)
}

func (l *kratosGormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	traceID := infralogging.TraceIDFromContext(ctx)
	l.helper.Warnw("trace_id", traceID, "msg", msg, "args", args)
}

func (l *kratosGormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	traceID := infralogging.TraceIDFromContext(ctx)
	l.helper.Errorw("trace_id", traceID, "msg", msg, "args", args)
}

func (l *kratosGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	traceID := infralogging.TraceIDFromContext(ctx)

	if err != nil {
		if l.ignoreRecordNotFound && errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		l.helper.Errorw(
			"trace_id", traceID,
			"latency", elapsed,
			"rows", rows,
			"sql", sql,
			"error", err,
		)
		return
	}

	if elapsed > l.slowThreshold {
		l.helper.Warnw(
			"trace_id", traceID,
			"latency", elapsed,
			"rows", rows,
			"sql", sql,
		)
		return
	}

	l.helper.Infow(
		"trace_id", traceID,
		"latency", elapsed,
		"rows", rows,
		"sql", sql,
	)
}
