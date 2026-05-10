package cache

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type CaptchaStore struct {
	client     *redis.Client
	keyPrefix  string
	expiration time.Duration
	timeout    time.Duration
}

func NewCaptchaStore(client *redis.Client, keyPrefix string, expiration time.Duration) *CaptchaStore {
	return &CaptchaStore{
		client:     client,
		keyPrefix:  keyPrefix,
		expiration: expiration,
		timeout:    3 * time.Second,
	}
}

func (s *CaptchaStore) Set(id string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	return s.client.Set(ctx, s.key(id), value, s.expiration).Err()
}

func (s *CaptchaStore) Get(id string, clear bool) string {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	key := s.key(id)
	value, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return ""
	}

	if clear {
		_ = s.client.Del(ctx, key).Err()
	}

	return value
}

func (s *CaptchaStore) Verify(id string, answer string, clear bool) bool {
	value := strings.TrimSpace(s.Get(id, clear))
	answer = strings.TrimSpace(answer)
	return strings.EqualFold(value, answer)
}

func (s *CaptchaStore) key(id string) string {
	return s.keyPrefix + id
}
