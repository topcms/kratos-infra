package mysql

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config struct {
	DSN                  string
	MaxIdleConns         int
	MaxOpenConns         int
	ConnMaxLifetime      time.Duration
	SlowThreshold        time.Duration
	IgnoreRecordNotFound bool
}

func NewDB(cfg Config, logger log.Logger) (*gorm.DB, error) {
	// 不设置 NamingStrategy.TablePrefix：表前缀 ts_ 由 gen 生成的 model.TableName() 显式返回；
	// 与 cmd/gen 中 WithModelNameStrategy 仅去掉「Go 类型名」前缀、不动物理表名一致。此处若再加 TablePrefix，易与默认复数规则叠加成错误表名（如 ts_users）。
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: newGormLogger(logger, cfg),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return db, nil
}
