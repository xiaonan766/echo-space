package cache

import (
	"testing"
	"time"
)

func TestLocalTokenBucketLimiterRefillsAfterWindow(t *testing.T) {
	limiter := NewLocalTokenBucketLimiter()
	now := time.Unix(100, 0)
	limiter.now = func() time.Time {
		return now
	}
	buckets := []DanmuRateBucket{{
		Key:      DanmuRateLimitKeyPrefix + "user:10001",
		Capacity: 1,
		Window:   100 * time.Millisecond,
	}}

	if !limiter.Allow(buckets) {
		t.Fatal("first request should be allowed")
	}
	if limiter.Allow(buckets) {
		t.Fatal("second request should be rejected after bucket is empty")
	}

	now = now.Add(100 * time.Millisecond)
	if !limiter.Allow(buckets) {
		t.Fatal("request should be allowed after token refill")
	}
}
