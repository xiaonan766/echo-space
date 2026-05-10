package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenStore struct {
	client    *redis.Client
	keyPrefix string
	timeout   time.Duration
}

func NewTokenStore(client *redis.Client, keyPrefix string) *TokenStore {
	return &TokenStore{
		client:    client,
		keyPrefix: keyPrefix,
		timeout:   3 * time.Second,
	}
}

func (s *TokenStore) Set(ctx context.Context, token string, value any, expiration time.Duration) error {
	content, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.client.Set(ctx, s.key(token), content, expiration).Err()
}

func (s *TokenStore) Delete(ctx context.Context, token string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.client.Del(ctx, s.key(token)).Err()
}

func (s *TokenStore) key(token string) string {
	return s.keyPrefix + token
}
