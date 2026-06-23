package cache

import (
	"context"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DanmuRateLimitKeyPrefix      = "echo-space:danmu:rate:"
	danmuRateLimitEvalTimeout    = 200 * time.Millisecond
	localDanmuBucketMaxIdleRatio = 2
)

const danmuRateLimitScript = `
local now = tonumber(ARGV[1])
local bucketCount = tonumber(ARGV[2])
local argIndex = 3
local nextTokens = {}
local windows = {}

for i = 1, bucketCount do
	local capacity = tonumber(ARGV[argIndex])
	local windowMs = tonumber(ARGV[argIndex + 1])
	argIndex = argIndex + 2

	if capacity <= 0 then
		capacity = 1
	end
	if windowMs <= 0 then
		windowMs = 1000
	end

	local tokens = capacity
	local lastRefillAt = now
	local storedTokens = redis.call("HGET", KEYS[i], "tokens")
	local storedRefillAt = redis.call("HGET", KEYS[i], "ts")

	if storedTokens and storedRefillAt then
		tokens = tonumber(storedTokens) or capacity
		lastRefillAt = tonumber(storedRefillAt) or now
		local elapsed = now - lastRefillAt
		if elapsed < 0 then
			elapsed = 0
		end
		tokens = math.min(capacity, tokens + elapsed * capacity / windowMs)
	end

	if tokens < 1 then
		return 0
	end

	nextTokens[i] = tokens - 1
	windows[i] = windowMs
end

for i = 1, bucketCount do
	redis.call("HMSET", KEYS[i], "tokens", nextTokens[i], "ts", now)
	redis.call("PEXPIRE", KEYS[i], math.max(windows[i] * 2, 1000))
end

return 1
`

type TokenBucketRule struct {
	Capacity int
	Window   time.Duration
}

type DanmuRateLimitConfig struct {
	User      TokenBucketRule
	UserVideo TokenBucketRule
	IP        TokenBucketRule
	Video     TokenBucketRule
}

type DanmuRateLimiter struct {
	redis   *redis.Client
	local   *LocalTokenBucketLimiter
	timeout time.Duration
}

type DanmuRateBucket struct {
	Key      string
	Capacity int
	Window   time.Duration
}

type LocalTokenBucketLimiter struct {
	mu      sync.Mutex
	buckets map[string]localTokenBucket
	checked int
	now     func() time.Time
}

type localTokenBucket struct {
	tokens       float64
	capacity     int
	window       time.Duration
	lastRefillAt time.Time
}

func NewDanmuRateLimiter(redisClient *redis.Client) *DanmuRateLimiter {
	return &DanmuRateLimiter{
		redis:   redisClient,
		local:   NewLocalTokenBucketLimiter(),
		timeout: danmuRateLimitEvalTimeout,
	}
}

func NewLocalTokenBucketLimiter() *LocalTokenBucketLimiter {
	return &LocalTokenBucketLimiter{
		buckets: make(map[string]localTokenBucket),
		now:     time.Now,
	}
}

func (l *DanmuRateLimiter) Allow(ctx context.Context, config DanmuRateLimitConfig, userID string, videoID string, clientIP string) (bool, error) {
	buckets := buildDanmuRateBuckets(config, userID, videoID, clientIP)
	if len(buckets) == 0 {
		return true, nil
	}

	if l != nil && l.redis != nil {
		allowed, err := l.allowByRedis(ctx, buckets)
		if err == nil {
			return allowed, nil
		}
		log.Printf("danmu redis rate limit failed, fallback to local: %v", err)
	}

	if l == nil || l.local == nil {
		return true, nil
	}
	return l.local.Allow(buckets), nil
}

func (l *DanmuRateLimiter) allowByRedis(ctx context.Context, buckets []DanmuRateBucket) (bool, error) {
	timeout := l.timeout
	if timeout <= 0 {
		timeout = danmuRateLimitEvalTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	keys := make([]string, 0, len(buckets))
	args := make([]any, 0, 2+len(buckets)*2)
	args = append(args, time.Now().UnixMilli(), len(buckets))
	for _, bucket := range buckets {
		rule := normalizeDanmuBucket(bucket)
		keys = append(keys, rule.Key)
		args = append(args, rule.Capacity, int64(rule.Window/time.Millisecond))
	}

	result, err := l.redis.Eval(ctx, danmuRateLimitScript, keys, args...).Result()
	if err != nil {
		return false, err
	}
	code, err := parseRedisInt(result)
	if err != nil {
		return false, err
	}
	return code == 1, nil
}

func (l *LocalTokenBucketLimiter) Allow(buckets []DanmuRateBucket) bool {
	if len(buckets) == 0 {
		return true
	}
	if l == nil {
		return true
	}

	now := time.Now()
	if l.now != nil {
		now = l.now()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	updated := make([]localTokenBucket, len(buckets))
	for index, bucket := range buckets {
		bucket = normalizeDanmuBucket(bucket)
		current := l.refilledBucket(bucket, now)
		if current.tokens < 1 {
			l.cleanup(now)
			return false
		}
		updated[index] = current
	}

	for index, bucket := range buckets {
		bucket = normalizeDanmuBucket(bucket)
		next := updated[index]
		next.tokens--
		next.lastRefillAt = now
		l.buckets[bucket.Key] = next
	}
	l.cleanup(now)
	return true
}

func (l *LocalTokenBucketLimiter) refilledBucket(bucket DanmuRateBucket, now time.Time) localTokenBucket {
	current, exists := l.buckets[bucket.Key]
	if !exists || current.capacity != bucket.Capacity || current.window != bucket.Window || current.lastRefillAt.IsZero() {
		return localTokenBucket{
			tokens:       float64(bucket.Capacity),
			capacity:     bucket.Capacity,
			window:       bucket.Window,
			lastRefillAt: now,
		}
	}

	elapsed := now.Sub(current.lastRefillAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > 0 {
		current.tokens = math.Min(float64(bucket.Capacity), current.tokens+float64(bucket.Capacity)*float64(elapsed)/float64(bucket.Window))
		current.lastRefillAt = now
	}
	return current
}

func (l *LocalTokenBucketLimiter) cleanup(now time.Time) {
	l.checked++
	if l.checked%128 != 0 {
		return
	}
	for key, bucket := range l.buckets {
		if bucket.window <= 0 {
			delete(l.buckets, key)
			continue
		}
		if now.Sub(bucket.lastRefillAt) > bucket.window*localDanmuBucketMaxIdleRatio {
			delete(l.buckets, key)
		}
	}
}

func buildDanmuRateBuckets(config DanmuRateLimitConfig, userID string, videoID string, clientIP string) []DanmuRateBucket {
	if clientIP == "" {
		clientIP = "unknown"
	}
	return []DanmuRateBucket{
		{Key: DanmuRateLimitKeyPrefix + "user:" + userID, Capacity: config.User.Capacity, Window: config.User.Window},
		{Key: DanmuRateLimitKeyPrefix + "user_video:" + userID + ":" + videoID, Capacity: config.UserVideo.Capacity, Window: config.UserVideo.Window},
		{Key: DanmuRateLimitKeyPrefix + "ip:" + clientIP, Capacity: config.IP.Capacity, Window: config.IP.Window},
		{Key: DanmuRateLimitKeyPrefix + "video:" + videoID, Capacity: config.Video.Capacity, Window: config.Video.Window},
	}
}

func normalizeDanmuBucket(bucket DanmuRateBucket) DanmuRateBucket {
	if bucket.Capacity <= 0 {
		bucket.Capacity = 1
	}
	if bucket.Window <= 0 {
		bucket.Window = time.Second
	}
	if bucket.Window < time.Millisecond {
		bucket.Window = time.Millisecond
	}
	if bucket.Key == "" {
		bucket.Key = DanmuRateLimitKeyPrefix + "empty:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return bucket
}
