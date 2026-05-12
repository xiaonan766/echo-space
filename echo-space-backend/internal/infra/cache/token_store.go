package cache

import (
	"context"
	"encoding/json"
	"time"
)

type TokenStore struct {
	cache     *HybridCache
	keyPrefix string
}

func NewTokenStore(cache *HybridCache, keyPrefix string) *TokenStore {
	return &TokenStore{
		cache:     cache,
		keyPrefix: keyPrefix,
	}
}

func (s *TokenStore) Set(ctx context.Context, token string, value any, expiration time.Duration) error {
	content, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.cache.Set(ctx, s.key(token), content, expiration, RecoverWriteBack)
}

func (s *TokenStore) Get(ctx context.Context, token string, dest any, localTTL time.Duration) (bool, error) {
	content, ok, err := s.cache.Get(ctx, s.key(token), localTTL, true)
	if err != nil || !ok {
		return ok, err
	}

	if err := json.Unmarshal(content, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (s *TokenStore) Delete(ctx context.Context, token string) error {
	return s.cache.Delete(ctx, s.key(token), RecoverWriteBack)
}

func (s *TokenStore) key(token string) string {
	return s.keyPrefix + token
}
