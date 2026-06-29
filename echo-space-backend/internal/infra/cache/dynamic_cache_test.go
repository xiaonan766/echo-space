package cache

import (
	"testing"
	"time"
)

func TestDynamicFeedRankTrimKeepsLatestThousand(t *testing.T) {
	if dynamicFeedRankTrimStop() != -1001 {
		t.Fatalf("rank trim stop = %d, want -1001", dynamicFeedRankTrimStop())
	}
}

func TestDynamicFeedTrimCutoffScoreUsesThirtyDays(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 999, time.Local)
	want := DynamicFeedScore(now.Add(-30 * 24 * time.Hour))
	if got := dynamicFeedTrimCutoffScore(now); got != want {
		t.Fatalf("cutoff score = %d, want %d", got, want)
	}
}
