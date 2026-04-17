package logger

import (
	"strings"

	kratoszap "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newZapLogger(cfg Config, meta ServiceMeta) (log.Logger, func(), error) {
	zapCfg := zap.NewProductionConfig()
	if strings.EqualFold(cfg.Format, "console") {
		zapCfg.Encoding = "console"
	}
	if strings.EqualFold(cfg.Format, "json") {
		zapCfg.Encoding = "json"
	}

	lvl := parseZapLevel(cfg.Level)
	zapCfg.Level = zap.NewAtomicLevelAt(lvl)
	zapCfg.DisableCaller = !cfg.Caller

	if strings.EqualFold(cfg.Output, "stderr") {
		zapCfg.OutputPaths = []string{"stderr"}
	} else {
		// default stdout to match containerized deployment habits.
		zapCfg.OutputPaths = []string{"stdout"}
	}
	zapCfg.ErrorOutputPaths = []string{"stderr"}

	zl, err := zapCfg.Build()
	if err != nil {
		return nil, nil, err
	}

	base := kratoszap.NewLogger(zl)
	logger := log.With(base,
		"service.id", meta.ID,
		"service.name", meta.Name,
		"service.version", meta.Version,
		"env", meta.Env,
	)

	cleanup := func() {
		_ = zl.Sync()
	}
	return logger, cleanup, nil
}

func parseZapLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zap.DebugLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "dpanic":
		return zap.DPanicLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	case "info":
		fallthrough
	default:
		return zap.InfoLevel
	}
}
