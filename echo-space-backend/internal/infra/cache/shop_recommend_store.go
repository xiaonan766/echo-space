package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const (
	hotPeripheralRecommendKey    = "echo-space:shop:recommend:peripheral:hot:v1"
	peripheralDetailKeyPrefix    = "echo-space:shop:peripheral:detail:"
	hotPeripheralDetailKeyPrefix = "echo-space:shop:hot:peripheral:detail:"
	hotPeripheralMarkerKeyPrefix = "echo-space:shop:hot:peripheral:"
	hotPeripheralScoreKeyPrefix  = "echo-space:shop:hot:score:peripheral:"

	shopRecommendTTL       = 5 * time.Minute
	peripheralDetailTTL    = 45 * time.Second
	hotPeripheralDetailTTL = 5 * time.Minute
	hotPeripheralMarkerTTL = 15 * time.Minute

	hotPeripheralWindowMinutes  = 5
	hotPeripheralVisitThreshold = 100
	hotPeripheralScoreTTL       = 10 * time.Minute
	hotPeripheralRedisTimeout   = 120 * time.Millisecond
)

type ShopRecommendStore struct {
	cache      *HybridCache
	redis      *redis.Client
	localMu    sync.Mutex
	localScore map[int64]map[uint64]int
}

func NewShopRecommendStore(cache *HybridCache, redisClient *redis.Client) *ShopRecommendStore {
	return &ShopRecommendStore{
		cache:      cache,
		redis:      redisClient,
		localScore: make(map[int64]map[uint64]int),
	}
}

func (s *ShopRecommendStore) GetHotPeripheral(ctx context.Context) ([]domain.WebShopItem, bool, error) {
	if s == nil || s.cache == nil {
		return nil, false, nil
	}

	content, ok, err := s.cache.Get(ctx, hotPeripheralRecommendKey, shopRecommendTTL, false)
	if err != nil || !ok {
		return nil, false, err
	}

	var list []domain.WebShopItem
	if err := json.Unmarshal(content, &list); err != nil {
		_ = s.cache.Delete(ctx, hotPeripheralRecommendKey, RecoverNone)
		return nil, false, err
	}
	return list, true, nil
}

func (s *ShopRecommendStore) SaveHotPeripheral(ctx context.Context, list []domain.WebShopItem) error {
	if s == nil || s.cache == nil {
		return nil
	}

	content, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, hotPeripheralRecommendKey, content, shopRecommendTTL, RecoverWriteBack)
}

func (s *ShopRecommendStore) SaveHotPeripheralToRedis(ctx context.Context, list []domain.WebShopItem) error {
	if s == nil || s.cache == nil {
		return nil
	}

	content, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return s.cache.SetRedis(ctx, hotPeripheralRecommendKey, content, shopRecommendTTL)
}

func (s *ShopRecommendStore) DeleteHotPeripheral(ctx context.Context) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.Delete(ctx, hotPeripheralRecommendKey, RecoverWriteBack)
}

func (s *ShopRecommendStore) GetPeripheralDetail(ctx context.Context, productID uint64, hot bool) (*domain.WebShopItem, bool, error) {
	if s == nil || s.cache == nil || productID == 0 {
		return nil, false, nil
	}

	if hot {
		if item, ok, err := s.getPeripheralDetailByKey(ctx, hotPeripheralDetailKey(productID), hotPeripheralDetailTTL); err != nil || ok {
			return item, ok, err
		}
		item, ok, err := s.getPeripheralDetailByKey(ctx, peripheralDetailKey(productID), peripheralDetailTTL)
		if err != nil || !ok {
			return nil, false, err
		}
		_ = s.SavePeripheralDetail(ctx, productID, item, true)
		return item, true, nil
	}
	return s.getPeripheralDetailByKey(ctx, peripheralDetailKey(productID), peripheralDetailTTL)
}

func (s *ShopRecommendStore) SavePeripheralDetail(ctx context.Context, productID uint64, item *domain.WebShopItem, hot bool) error {
	if s == nil || s.cache == nil || productID == 0 || item == nil {
		return nil
	}

	content, err := json.Marshal(item)
	if err != nil {
		return err
	}

	key := peripheralDetailKey(productID)
	ttl := peripheralDetailTTL
	if hot {
		key = hotPeripheralDetailKey(productID)
		ttl = hotPeripheralDetailTTL
	}
	return s.cache.Set(ctx, key, content, ttl, RecoverWriteBack)
}

func (s *ShopRecommendStore) SavePeripheralDetailToRedis(ctx context.Context, productID uint64, item *domain.WebShopItem, hot bool) error {
	if s == nil || s.cache == nil || productID == 0 || item == nil {
		return nil
	}

	content, err := json.Marshal(item)
	if err != nil {
		return err
	}

	key := peripheralDetailKey(productID)
	ttl := peripheralDetailTTL
	if hot {
		key = hotPeripheralDetailKey(productID)
		ttl = hotPeripheralDetailTTL
	}
	return s.cache.SetRedis(ctx, key, content, ttl)
}

func (s *ShopRecommendStore) DeletePeripheralDetail(ctx context.Context, productID uint64) error {
	if s == nil || s.cache == nil || productID == 0 {
		return nil
	}

	if err := s.cache.Delete(ctx, peripheralDetailKey(productID), RecoverWriteBack); err != nil {
		return err
	}
	return s.cache.Delete(ctx, hotPeripheralDetailKey(productID), RecoverWriteBack)
}

func (s *ShopRecommendStore) DeletePeripheralDetailFromRedis(ctx context.Context, productID uint64) error {
	if s == nil || s.cache == nil || productID == 0 {
		return nil
	}

	if err := s.cache.DeleteRedis(ctx, peripheralDetailKey(productID)); err != nil {
		return err
	}
	return s.cache.DeleteRedis(ctx, hotPeripheralDetailKey(productID))
}

func (s *ShopRecommendStore) IsHotPeripheral(ctx context.Context, productID uint64) bool {
	if s == nil || s.cache == nil || productID == 0 {
		return false
	}

	_, ok, err := s.cache.Get(ctx, hotPeripheralMarkerKey(productID), hotPeripheralMarkerTTL, false)
	return err == nil && ok
}

func (s *ShopRecommendStore) MarkHotPeripheral(ctx context.Context, productID uint64) error {
	if s == nil || s.cache == nil || productID == 0 {
		return nil
	}
	return s.cache.Set(ctx, hotPeripheralMarkerKey(productID), []byte("1"), hotPeripheralMarkerTTL, RecoverWriteBack)
}

func (s *ShopRecommendStore) RestoreHotPeripheralMarker(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s == nil || s.cache == nil || key == "" || len(value) == 0 {
		return nil
	}
	if !strings.HasPrefix(key, hotPeripheralMarkerKeyPrefix) || strings.HasPrefix(key, hotPeripheralDetailKeyPrefix) {
		return nil
	}
	return s.cache.Set(ctx, key, value, ttl, RecoverWriteBack)
}

func (s *ShopRecommendStore) RestoreHotPeripheralMarkerToRedis(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s == nil || s.cache == nil || key == "" || len(value) == 0 {
		return nil
	}
	if !strings.HasPrefix(key, hotPeripheralMarkerKeyPrefix) || strings.HasPrefix(key, hotPeripheralDetailKeyPrefix) {
		return nil
	}
	return s.cache.SetRedis(ctx, key, value, ttl)
}

func (s *ShopRecommendStore) DeleteHotPeripheralMarker(ctx context.Context, productID uint64) error {
	if s == nil || s.cache == nil || productID == 0 {
		return nil
	}
	return s.cache.Delete(ctx, hotPeripheralMarkerKey(productID), RecoverWriteBack)
}

func (s *ShopRecommendStore) DeleteShopCacheKeyFromRedis(ctx context.Context, key string) error {
	if s == nil || s.cache == nil || !isShopCacheRecoveryKey(key) {
		return nil
	}
	return s.cache.DeleteRedis(ctx, key)
}

func (s *ShopRecommendStore) TrackPeripheralVisit(ctx context.Context, productID uint64) bool {
	if s == nil || productID == 0 {
		return false
	}

	if total, ok := s.trackPeripheralVisitInRedis(ctx, productID); ok {
		if total >= hotPeripheralVisitThreshold {
			_ = s.MarkHotPeripheral(ctx, productID)
			return true
		}
		return false
	}

	total := s.trackPeripheralVisitInLocal(productID)
	if total >= hotPeripheralVisitThreshold {
		_ = s.MarkHotPeripheral(ctx, productID)
		return true
	}
	return false
}

func (s *ShopRecommendStore) getPeripheralDetailByKey(ctx context.Context, key string, ttl time.Duration) (*domain.WebShopItem, bool, error) {
	content, ok, err := s.cache.Get(ctx, key, ttl, false)
	if err != nil || !ok {
		return nil, false, err
	}

	var item domain.WebShopItem
	if err := json.Unmarshal(content, &item); err != nil {
		_ = s.cache.Delete(ctx, key, RecoverNone)
		return nil, false, err
	}
	return &item, true, nil
}

func (s *ShopRecommendStore) trackPeripheralVisitInRedis(ctx context.Context, productID uint64) (int, bool) {
	if s.redis == nil {
		return 0, false
	}

	ctx, cancel := context.WithTimeout(ctx, hotPeripheralRedisTimeout)
	defer cancel()

	now := time.Now()
	member := strconv.FormatUint(productID, 10)
	scoreKey := hotPeripheralScoreKey(now)

	pipe := s.redis.Pipeline()
	pipe.ZIncrBy(ctx, scoreKey, 1, member)
	pipe.Expire(ctx, scoreKey, hotPeripheralScoreTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, false
	}

	scorePipe := s.redis.Pipeline()
	scoreCommands := make([]*redis.FloatCmd, 0, hotPeripheralWindowMinutes)
	for index := 0; index < hotPeripheralWindowMinutes; index++ {
		key := hotPeripheralScoreKey(now.Add(-time.Duration(index) * time.Minute))
		scoreCommands = append(scoreCommands, scorePipe.ZScore(ctx, key, member))
	}
	if _, err := scorePipe.Exec(ctx); err != nil && err != redis.Nil {
		return 0, false
	}

	total := 0
	for _, command := range scoreCommands {
		value, err := command.Result()
		if err != nil && err != redis.Nil {
			return 0, false
		}
		total += int(math.Round(value))
	}
	return total, true
}

func (s *ShopRecommendStore) trackPeripheralVisitInLocal(productID uint64) int {
	nowMinute := time.Now().Truncate(time.Minute).Unix()

	s.localMu.Lock()
	defer s.localMu.Unlock()

	bucket, ok := s.localScore[nowMinute]
	if !ok {
		bucket = make(map[uint64]int)
		s.localScore[nowMinute] = bucket
	}
	bucket[productID]++

	oldestMinute := nowMinute - int64(hotPeripheralScoreTTL/time.Minute)*60
	for minute := range s.localScore {
		if minute < oldestMinute {
			delete(s.localScore, minute)
		}
	}

	total := 0
	for index := 0; index < hotPeripheralWindowMinutes; index++ {
		minute := nowMinute - int64(index)*60
		total += s.localScore[minute][productID]
	}
	return total
}

func hotPeripheralScoreKey(value time.Time) string {
	return hotPeripheralScoreKeyPrefix + value.Format("200601021504")
}

func peripheralDetailKey(productID uint64) string {
	return fmt.Sprintf("%s%d", peripheralDetailKeyPrefix, productID)
}

func hotPeripheralDetailKey(productID uint64) string {
	return fmt.Sprintf("%s%d", hotPeripheralDetailKeyPrefix, productID)
}

func hotPeripheralMarkerKey(productID uint64) string {
	return fmt.Sprintf("%s%d", hotPeripheralMarkerKeyPrefix, productID)
}

func isShopCacheRecoveryKey(key string) bool {
	return key == hotPeripheralRecommendKey ||
		strings.HasPrefix(key, peripheralDetailKeyPrefix) ||
		strings.HasPrefix(key, hotPeripheralDetailKeyPrefix) ||
		strings.HasPrefix(key, hotPeripheralMarkerKeyPrefix)
}
