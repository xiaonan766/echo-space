package cache

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const (
	defaultVideoHotMetricsKeyPrefix = "echo-space:video:hot:metrics:"
	defaultVideoHotRankKey          = "echo-space:video:hot:rank"
	defaultVideoHotPlayDedupeTTL    = 30 * time.Minute
	videoHotEventKeyPrefix          = "echo-space:video:hot:event:"
	videoHotPlayDedupeKeyPrefix     = "echo-space:video:play:dedupe:"
	videoHotEventTTL                = 7 * 24 * time.Hour
)

const incrementVideoHotMetricScript = `
local metricsKey = KEYS[1]
local field = ARGV[1]
local delta = tonumber(ARGV[2])

if delta ~= 0 then
	local current = redis.call("HINCRBY", metricsKey, field, delta)
	if current < 0 then
		redis.call("HSET", metricsKey, field, 0)
	end
end

local playCount = redis.call("HGET", metricsKey, "playCount") or "0"
local likeCount = redis.call("HGET", metricsKey, "likeCount") or "0"
local collectCount = redis.call("HGET", metricsKey, "collectCount") or "0"
local coinCount = redis.call("HGET", metricsKey, "coinCount") or "0"
local commentCount = redis.call("HGET", metricsKey, "commentCount") or "0"
return {playCount, likeCount, collectCount, coinCount, commentCount}
`

type VideoHotStore struct {
	redis            *redis.Client
	metricsKeyPrefix string
	rankKey          string
	playDedupeTTL    time.Duration
}

func NewVideoHotStore(redisClient *redis.Client, cfg config.VideoHotConfig) *VideoHotStore {
	metricsKeyPrefix := strings.TrimSpace(cfg.MetricsKeyPrefix)
	if metricsKeyPrefix == "" {
		metricsKeyPrefix = defaultVideoHotMetricsKeyPrefix
	}
	rankKey := strings.TrimSpace(cfg.RankKey)
	if rankKey == "" {
		rankKey = defaultVideoHotRankKey
	}
	playDedupeTTL := cfg.PlayDedupeTTLDuration()
	if playDedupeTTL <= 0 {
		playDedupeTTL = defaultVideoHotPlayDedupeTTL
	}
	return &VideoHotStore{
		redis:            redisClient,
		metricsKeyPrefix: metricsKeyPrefix,
		rankKey:          rankKey,
		playDedupeTTL:    playDedupeTTL,
	}
}

func (s *VideoHotStore) MarkPlayStart(ctx context.Context, videoID string, deviceID string) (bool, error) {
	if s == nil || s.redis == nil {
		return true, nil
	}
	videoID = strings.TrimSpace(videoID)
	deviceID = strings.TrimSpace(deviceID)
	if videoID == "" || deviceID == "" {
		return true, nil
	}
	return s.redis.SetNX(ctx, videoHotPlayDedupeKey(videoID, deviceID), "1", s.playDedupeTTL).Result()
}

func (s *VideoHotStore) BeginMetricEvent(ctx context.Context, eventID string) (bool, error) {
	if s == nil || s.redis == nil {
		return true, nil
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return true, nil
	}
	return s.redis.SetNX(ctx, videoHotEventKey(eventID), "1", videoHotEventTTL).Result()
}

func (s *VideoHotStore) ReleaseMetricEvent(ctx context.Context, eventID string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil
	}
	return s.redis.Del(ctx, videoHotEventKey(eventID)).Err()
}

func (s *VideoHotStore) IncrementMetric(ctx context.Context, event domain.VideoHotMetricEvent) (domain.VideoHotMetrics, error) {
	if s == nil || s.redis == nil {
		return domain.VideoHotMetrics{VideoID: event.VideoID}, nil
	}
	field, ok := videoHotMetricField(event.EventType)
	if !ok {
		return domain.VideoHotMetrics{VideoID: event.VideoID}, fmt.Errorf("unsupported video hot metric event type %s", event.EventType)
	}

	result, err := s.redis.Eval(ctx, incrementVideoHotMetricScript, []string{s.metricsKey(event.VideoID)}, field, event.Delta).Result()
	if err != nil {
		return domain.VideoHotMetrics{}, err
	}
	return parseVideoHotMetrics(event.VideoID, result)
}

func (s *VideoHotStore) SetMetrics(ctx context.Context, metrics domain.VideoHotMetrics) error {
	if s == nil || s.redis == nil {
		return nil
	}
	metrics = normalizeVideoHotMetrics(metrics)
	return s.redis.HSet(ctx, s.metricsKey(metrics.VideoID), map[string]any{
		"playCount":    metrics.PlayCount,
		"likeCount":    metrics.LikeCount,
		"collectCount": metrics.CollectCount,
		"coinCount":    metrics.CoinCount,
		"commentCount": metrics.CommentCount,
	}).Err()
}

func (s *VideoHotStore) GetMetrics(ctx context.Context, videoID string) (domain.VideoHotMetrics, error) {
	if s == nil || s.redis == nil {
		return domain.VideoHotMetrics{VideoID: videoID}, nil
	}
	values, err := s.redis.HMGet(ctx, s.metricsKey(videoID), "playCount", "likeCount", "collectCount", "coinCount", "commentCount").Result()
	if err != nil {
		return domain.VideoHotMetrics{}, err
	}
	return parseVideoHotMetricValues(videoID, values)
}

func (s *VideoHotStore) SetRank(ctx context.Context, videoID string, heatScore int64) error {
	if s == nil || s.redis == nil {
		return nil
	}
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return nil
	}
	if heatScore <= 0 {
		return s.redis.ZRem(ctx, s.rankKey, videoID).Err()
	}
	return s.redis.ZAdd(ctx, s.rankKey, redis.Z{
		Score:  float64(heatScore),
		Member: videoID,
	}).Err()
}

func (s *VideoHotStore) ListRanks(ctx context.Context, pageNo int, pageSize int) ([]domain.VideoHotRankEntry, int64, error) {
	if s == nil || s.redis == nil {
		return []domain.VideoHotRankEntry{}, 0, nil
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	totalCount, err := s.redis.ZCard(ctx, s.rankKey).Result()
	if err != nil {
		return nil, 0, err
	}
	if totalCount == 0 {
		return []domain.VideoHotRankEntry{}, 0, nil
	}

	start := int64((pageNo - 1) * pageSize)
	stop := start + int64(pageSize) - 1
	values, err := s.redis.ZRevRangeWithScores(ctx, s.rankKey, start, stop).Result()
	if err != nil {
		return nil, 0, err
	}

	list := make([]domain.VideoHotRankEntry, 0, len(values))
	for index, item := range values {
		videoID := strings.TrimSpace(fmt.Sprint(item.Member))
		if videoID == "" {
			continue
		}
		list = append(list, domain.VideoHotRankEntry{
			VideoID:   videoID,
			Rank:      int(start) + index + 1,
			HeatScore: int64(math.Round(item.Score)),
		})
	}
	return list, totalCount, nil
}

func (s *VideoHotStore) metricsKey(videoID string) string {
	return s.metricsKeyPrefix + strings.TrimSpace(videoID)
}

func videoHotPlayDedupeKey(videoID string, deviceID string) string {
	return videoHotPlayDedupeKeyPrefix + strings.TrimSpace(videoID) + ":" + strings.TrimSpace(deviceID)
}

func videoHotEventKey(eventID string) string {
	return videoHotEventKeyPrefix + strings.TrimSpace(eventID)
}

func videoHotMetricField(eventType string) (string, bool) {
	switch strings.TrimSpace(eventType) {
	case domain.VideoHotMetricEventPlay:
		return "playCount", true
	case domain.VideoHotMetricEventLike:
		return "likeCount", true
	case domain.VideoHotMetricEventCollect:
		return "collectCount", true
	case domain.VideoHotMetricEventCoin:
		return "coinCount", true
	case domain.VideoHotMetricEventComment:
		return "commentCount", true
	default:
		return "", false
	}
}

func parseVideoHotMetrics(videoID string, result any) (domain.VideoHotMetrics, error) {
	values, ok := result.([]any)
	if !ok {
		return domain.VideoHotMetrics{}, fmt.Errorf("unexpected video hot metric result type %T", result)
	}
	return parseVideoHotMetricValues(videoID, values)
}

func parseVideoHotMetricValues(videoID string, values []any) (domain.VideoHotMetrics, error) {
	metrics := domain.VideoHotMetrics{VideoID: strings.TrimSpace(videoID)}
	if len(values) != 5 {
		return metrics, fmt.Errorf("unexpected video hot metric value count %d", len(values))
	}
	var err error
	metrics.PlayCount, err = parseOptionalRedisInt(values[0])
	if err != nil {
		return metrics, err
	}
	metrics.LikeCount, err = parseOptionalRedisInt(values[1])
	if err != nil {
		return metrics, err
	}
	metrics.CollectCount, err = parseOptionalRedisInt(values[2])
	if err != nil {
		return metrics, err
	}
	metrics.CoinCount, err = parseOptionalRedisInt(values[3])
	if err != nil {
		return metrics, err
	}
	metrics.CommentCount, err = parseOptionalRedisInt(values[4])
	if err != nil {
		return metrics, err
	}
	return normalizeVideoHotMetrics(metrics), nil
}

func parseOptionalRedisInt(value any) (int, error) {
	if value == nil {
		return 0, nil
	}
	return parseRedisInt(value)
}

func normalizeVideoHotMetrics(metrics domain.VideoHotMetrics) domain.VideoHotMetrics {
	metrics.VideoID = strings.TrimSpace(metrics.VideoID)
	if metrics.PlayCount < 0 {
		metrics.PlayCount = 0
	}
	if metrics.LikeCount < 0 {
		metrics.LikeCount = 0
	}
	if metrics.CollectCount < 0 {
		metrics.CollectCount = 0
	}
	if metrics.CoinCount < 0 {
		metrics.CoinCount = 0
	}
	if metrics.CommentCount < 0 {
		metrics.CommentCount = 0
	}
	return metrics
}

func LogVideoHotStoreError(operation string, err error) {
	if err != nil {
		log.Printf("video hot cache %s failed: %v", operation, err)
	}
}
