package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
)

func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect redis %s: %w", cfg.Addr, err)
	}

	return client, nil
}
