package cache

import (
	"context"
	"strings"
	"time"
)

type CaptchaStore struct {
	cache      *HybridCache
	keyPrefix  string
	expiration time.Duration
}

func NewCaptchaStore(cache *HybridCache, keyPrefix string, expiration time.Duration) *CaptchaStore {
	return &CaptchaStore{
		cache:      cache,
		keyPrefix:  keyPrefix,
		expiration: expiration,
	}
}

func (s *CaptchaStore) Set(id string, value string) error {
	return s.cache.Set(context.Background(), s.key(id), []byte(value), s.expiration, RecoverNone)
}

func (s *CaptchaStore) Get(id string, clear bool) string {
	key := s.key(id)
	value, ok, err := s.cache.Get(context.Background(), key, s.expiration, true)
	if err != nil || !ok {
		return ""
	}

	if clear {
		_ = s.cache.Delete(context.Background(), key, RecoverNone)
	}

	return string(value)
}

func (s *CaptchaStore) Verify(id string, answer string, clear bool) bool {
	value := strings.TrimSpace(s.Get(id, clear))
	answer = strings.TrimSpace(answer)
	return strings.EqualFold(value, answer)
}

func (s *CaptchaStore) key(id string) string {
	return s.keyPrefix + id
}
