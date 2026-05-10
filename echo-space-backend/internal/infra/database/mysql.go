package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
)

func NewMySQLClient(ctx context.Context, cfg config.MySQLConfig) (*gorm.DB, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("mysql dsn is empty")
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get mysql sql db: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	return db, nil
}
