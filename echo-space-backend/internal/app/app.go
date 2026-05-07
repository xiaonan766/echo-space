package app

import (
	"context"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	approuter "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/router"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

type App struct {
	cfg    config.Config
	redis  *redis.Client
	router http.Handler
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}

	router := approuter.New(approuter.Dependencies{
		Config: cfg,
		Redis:  redisClient,
	})

	return &App{
		cfg:    cfg,
		redis:  redisClient,
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
}
