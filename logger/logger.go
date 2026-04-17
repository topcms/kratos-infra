package logger

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

type Config struct {
	Driver string
	Level  string
	Format string
	Output string
	Caller bool
}

type ServiceMeta struct {
	ID      string
	Name    string
	Version string
	Env     string
}

type builder func(cfg Config, meta ServiceMeta) (log.Logger, func(), error)

var builders = map[string]builder{
	"zap": newZapLogger,
}

func New(cfg Config, meta ServiceMeta) (log.Logger, func(), error) {
	driver := cfg.Driver
	if driver == "" {
		driver = "zap"
	}

	b, ok := builders[driver]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported log driver: %s", driver)
	}
	return b(cfg, meta)
}
