package cache

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RecoverPolicy int

const (
	RecoverNone RecoverPolicy = iota
	RecoverWriteBack
)

type RecoveryHandler interface {
	HandleDirtyWrite(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	HandlePendingDelete(ctx context.Context, key string) (bool, error)
}

type HybridCache struct {
	redisClient  *redis.Client
	localCache   *LocalCache
	redisTimeout time.Duration
	retryAfter   time.Duration
	pingInterval time.Duration

	stateMu       sync.RWMutex
	redisUp       bool
	nextRedisTry  time.Time
	dirtyWrites   map[string]dirtyWrite
	pendingDelete map[string]time.Time

	recoveryMu      sync.RWMutex
	recoveryHandler RecoveryHandler

	stopCh chan struct{}
	once   sync.Once
}

type dirtyWrite struct {
	value     []byte
	expireAt  time.Time
	updatedAt time.Time
}

func NewHybridCache(redisClient *redis.Client) *HybridCache {
	cache := &HybridCache{
		redisClient:   redisClient,
		localCache:    NewLocalCache(),
		redisTimeout:  500 * time.Millisecond,
		retryAfter:    3 * time.Second,
		pingInterval:  5 * time.Second,
		redisUp:       true,
		dirtyWrites:   make(map[string]dirtyWrite),
		pendingDelete: make(map[string]time.Time),
		stopCh:        make(chan struct{}),
	}
	go cache.watchRedis()
	return cache
}

func (c *HybridCache) Close() {
	c.once.Do(func() {
		close(c.stopCh)
	})
}

func (c *HybridCache) SetRecoveryHandler(handler RecoveryHandler) {
	c.recoveryMu.Lock()
	defer c.recoveryMu.Unlock()
	c.recoveryHandler = handler
}

func (c *HybridCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration, policy RecoverPolicy) error {
	c.localCache.Set(key, value, ttl)

	if !c.shouldTryRedis() {
		c.markWriteBackIfNeeded(key, value, ttl, policy)
		return nil
	}

	if err := c.redisSet(ctx, key, value, ttl); err != nil {
		c.markRedisFailure(err)
		c.markWriteBackIfNeeded(key, value, ttl, policy)
		return nil
	}

	c.markRedisSuccess()
	c.clearDirty(key)
	return nil
}

func (c *HybridCache) Get(ctx context.Context, key string, localTTL time.Duration, fallbackOnRedisMiss bool) ([]byte, bool, error) {
	if c.shouldTryRedis() {
		value, err := c.redisGet(ctx, key)
		if err == nil {
			c.markRedisSuccess()
			c.localCache.Set(key, value, localTTL)
			return value, true, nil
		}
		if err == redis.Nil {
			c.markRedisSuccess()
			if fallbackOnRedisMiss {
				return c.getLocal(key)
			}
			return nil, false, nil
		}

		c.markRedisFailure(err)
	}

	return c.getLocal(key)
}

func (c *HybridCache) Delete(ctx context.Context, key string, policy RecoverPolicy) error {
	c.localCache.Delete(key)

	if !c.shouldTryRedis() {
		c.markPendingDeleteIfNeeded(key, policy)
		return nil
	}

	if err := c.redisDelete(ctx, key); err != nil {
		c.markRedisFailure(err)
		c.markPendingDeleteIfNeeded(key, policy)
		return nil
	}

	c.markRedisSuccess()
	c.clearDirty(key)
	return nil
}

func (c *HybridCache) SetRedis(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil {
		return errors.New("hybrid cache is nil")
	}
	if err := c.redisSet(ctx, key, value, ttl); err != nil {
		c.markRedisFailure(err)
		return err
	}
	c.localCache.Set(key, value, ttl)
	c.markRedisSuccess()
	c.clearDirty(key)
	return nil
}

func (c *HybridCache) DeleteRedis(ctx context.Context, key string) error {
	if c == nil {
		return errors.New("hybrid cache is nil")
	}
	if err := c.redisDelete(ctx, key); err != nil {
		c.markRedisFailure(err)
		return err
	}
	c.localCache.Delete(key)
	c.markRedisSuccess()
	c.clearDirty(key)
	return nil
}

func (c *HybridCache) getLocal(key string) ([]byte, bool, error) {
	value, ok := c.localCache.Get(key)
	return value, ok, nil
}

func (c *HybridCache) redisSet(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, c.redisTimeout)
	defer cancel()
	return c.redisClient.Set(ctx, key, value, ttl).Err()
}

func (c *HybridCache) redisGet(ctx context.Context, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.redisTimeout)
	defer cancel()
	return c.redisClient.Get(ctx, key).Bytes()
}

func (c *HybridCache) redisDelete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, c.redisTimeout)
	defer cancel()
	return c.redisClient.Del(ctx, key).Err()
}

func (c *HybridCache) shouldTryRedis() bool {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.redisUp || time.Now().After(c.nextRedisTry)
}

func (c *HybridCache) markRedisFailure(err error) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.redisUp {
		log.Printf("redis unavailable, using local cache fallback: %v", err)
	}
	c.redisUp = false
	c.nextRedisTry = time.Now().Add(c.retryAfter)
}

func (c *HybridCache) markRedisSuccess() {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if !c.redisUp {
		log.Printf("redis recovered, syncing local cache changes")
	}
	c.redisUp = true
	c.nextRedisTry = time.Time{}
}

func (c *HybridCache) markWriteBackIfNeeded(key string, value []byte, ttl time.Duration, policy RecoverPolicy) {
	if policy != RecoverWriteBack {
		return
	}

	var expireAt time.Time
	if ttl > 0 {
		expireAt = time.Now().Add(ttl)
	}

	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	delete(c.pendingDelete, key)
	c.dirtyWrites[key] = dirtyWrite{
		value:     append([]byte(nil), value...),
		expireAt:  expireAt,
		updatedAt: time.Now(),
	}
}

func (c *HybridCache) markPendingDeleteIfNeeded(key string, policy RecoverPolicy) {
	if policy != RecoverWriteBack {
		return
	}

	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	delete(c.dirtyWrites, key)
	c.pendingDelete[key] = time.Now()
}

func (c *HybridCache) clearDirty(key string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	delete(c.dirtyWrites, key)
	delete(c.pendingDelete, key)
}

func (c *HybridCache) watchRedis() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.pingAndRecover()
		case <-c.stopCh:
			return
		}
	}
}

func (c *HybridCache) pingAndRecover() {
	ctx, cancel := context.WithTimeout(context.Background(), c.redisTimeout)
	defer cancel()

	if err := c.redisClient.Ping(ctx).Err(); err != nil {
		c.markRedisFailure(err)
		return
	}

	c.markRedisSuccess()
	c.flushDirty()
}

func (c *HybridCache) flushDirty() {
	deletes, writes := c.snapshotDirty()
	handler := c.getRecoveryHandler()

	for key, updatedAt := range deletes {
		if handler != nil {
			handled, err := handler.HandlePendingDelete(context.Background(), key)
			if err != nil {
				log.Printf("publish cache delete recovery task failed, fallback to redis delete: key=%s err=%v", key, err)
			}
			if err == nil && handled {
				c.removePendingDelete(key, updatedAt)
				continue
			}
		}

		if err := c.redisDelete(context.Background(), key); err != nil {
			c.markRedisFailure(err)
			return
		}
		c.removePendingDelete(key, updatedAt)
	}

	for key, item := range writes {
		ttl, ok := item.remainingTTL(time.Now())
		if !ok {
			c.removeDirtyWrite(key, item.updatedAt)
			continue
		}

		if handler != nil {
			handled, err := handler.HandleDirtyWrite(context.Background(), key, item.value, ttl)
			if err != nil {
				log.Printf("publish cache recovery task failed, fallback to redis write: key=%s err=%v", key, err)
			}
			if err == nil && handled {
				c.removeDirtyWrite(key, item.updatedAt)
				continue
			}
		}

		if err := c.redisSet(context.Background(), key, item.value, ttl); err != nil {
			c.markRedisFailure(err)
			return
		}
		c.removeDirtyWrite(key, item.updatedAt)
	}
}

func (c *HybridCache) getRecoveryHandler() RecoveryHandler {
	c.recoveryMu.RLock()
	defer c.recoveryMu.RUnlock()
	return c.recoveryHandler
}

func (c *HybridCache) snapshotDirty() (map[string]time.Time, map[string]dirtyWrite) {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()

	deletes := make(map[string]time.Time, len(c.pendingDelete))
	for key, updatedAt := range c.pendingDelete {
		deletes[key] = updatedAt
	}

	writes := make(map[string]dirtyWrite, len(c.dirtyWrites))
	for key, item := range c.dirtyWrites {
		writes[key] = dirtyWrite{
			value:     append([]byte(nil), item.value...),
			expireAt:  item.expireAt,
			updatedAt: item.updatedAt,
		}
	}
	return deletes, writes
}

func (c *HybridCache) removePendingDelete(key string, updatedAt time.Time) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	current, ok := c.pendingDelete[key]
	if ok && current.Equal(updatedAt) {
		delete(c.pendingDelete, key)
	}
}

func (c *HybridCache) removeDirtyWrite(key string, updatedAt time.Time) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	current, ok := c.dirtyWrites[key]
	if ok && current.updatedAt.Equal(updatedAt) {
		delete(c.dirtyWrites, key)
	}
}

func (w dirtyWrite) remainingTTL(now time.Time) (time.Duration, bool) {
	if w.expireAt.IsZero() {
		return 0, true
	}

	if !w.expireAt.After(now) {
		return 0, false
	}
	return w.expireAt.Sub(now), true
}
