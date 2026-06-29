package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	searchKeywordKey          = "echo-space:search:keyword:score"
	searchKeywordRedisTimeout = 120 * time.Millisecond
)

type SearchKeywordStore struct {
	redis      *redis.Client
	localMu    sync.Mutex
	localScore map[string]float64
}

func NewSearchKeywordStore(redisClient *redis.Client) *SearchKeywordStore {
	return &SearchKeywordStore{
		redis:      redisClient,
		localScore: make(map[string]float64),
	}
}

func (s *SearchKeywordStore) Add(ctx context.Context, keyword string) error {
	keyword = strings.TrimSpace(keyword)
	if s == nil || keyword == "" {
		return nil
	}

	if s.redis == nil {
		s.addLocal(keyword)
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, searchKeywordRedisTimeout)
	defer cancel()

	if err := s.redis.ZIncrBy(timeoutCtx, searchKeywordKey, 1, keyword).Err(); err != nil {
		s.addLocal(keyword)
		return err
	}
	return nil
}

func (s *SearchKeywordStore) addLocal(keyword string) {
	s.localMu.Lock()
	defer s.localMu.Unlock()
	s.localScore[keyword]++
}
