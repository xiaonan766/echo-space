package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	approuter "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/router"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/database"
)

type App struct {
	cfg    config.Config
	redis  *redis.Client
	db     *gorm.DB
	router http.Handler
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}

	mysqlDB, err := database.NewMySQLClient(ctx, cfg.MySQL)
	if err != nil {
		_ = redisClient.Close()
		return nil, err
	}

	if cfg.MySQL.AutoMigrate {
		if err := mysqlDB.AutoMigrate(&domain.CategoryInfo{}); err != nil {
			_ = redisClient.Close()
			closeDB(mysqlDB)
			return nil, fmt.Errorf("auto migrate mysql: %w", err)
		}
	}

	router := approuter.New(approuter.Dependencies{
		Config: cfg,
		Redis:  redisClient,
		DB:     mysqlDB,
	})

	return &App{
		cfg:    cfg,
		redis:  redisClient,
		db:     mysqlDB,
		router: router,
	}, nil
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) Close() {
	if a.redis != nil {
		_ = a.redis.Close()
	}
	closeDB(a.db)
}

func closeDB(db *gorm.DB) {
	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}
