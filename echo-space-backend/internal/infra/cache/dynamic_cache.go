package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	dynamicActiveUsersKey    = "echo-space:dynamic:active-users"
	dynamicFeedImagePrefix   = "image:"
	dynamicAuthorPolicyTTL   = 5 * time.Minute
	dynamicFeedTTL           = 8 * 24 * time.Hour
	dynamicFeedMaxAge        = 30 * 24 * time.Hour
	dynamicFeedMaxCount      = 1000
	dynamicCacheTimeout      = 120 * time.Millisecond
	dynamicActiveUsersMaxAge = 30 * 24 * time.Hour
	dynamicFeedScanBatchSize = 100
	dynamicFeedWriteBatch    = 500
)

type DynamicAuthorPolicy struct {
	AuthorUserID    string `json:"authorUserId"`
	FansCount       int64  `json:"fansCount"`
	ReadExpansion   bool   `json:"readExpansion"`
	CalculatedAtSec int64  `json:"calculatedAtSec"`
}

type DynamicFeedCacheItem struct {
	ContentType  int
	ContentID    string
	VideoID      string
	AuthorUserID string
	Score        int64
}

type DynamicFeedCacheCursor struct {
	HasCursor   bool
	Score       int64
	ContentType int
	ContentID   string
	VideoID     string
}

type DynamicFeedCachePage struct {
	Hit     bool
	Items   []DynamicFeedCacheItem
	HasMore bool
}

type DynamicActiveStore struct {
	redis *redis.Client
}

func NewDynamicActiveStore(redisClient *redis.Client) *DynamicActiveStore {
	return &DynamicActiveStore{redis: redisClient}
}

func (s *DynamicActiveStore) MarkActive(ctx context.Context, userID string, activeAt time.Time) error {
	userID = strings.TrimSpace(userID)
	if s == nil || s.redis == nil || userID == "" {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dynamicCacheTimeout)
	defer cancel()

	pipe := s.redis.Pipeline()
	pipe.ZAdd(timeoutCtx, dynamicActiveUsersKey, redis.Z{
		Score:  float64(activeAt.Unix()),
		Member: userID,
	})
	pipe.ZRemRangeByScore(timeoutCtx, dynamicActiveUsersKey, "-inf", strconv.FormatInt(activeAt.Add(-dynamicActiveUsersMaxAge).Unix(), 10))
	_, err := pipe.Exec(timeoutCtx)
	return err
}

func (s *DynamicActiveStore) ListActiveUserIDs(ctx context.Context, since time.Time, limit int64) ([]string, error) {
	if s == nil || s.redis == nil {
		return []string{}, nil
	}
	if limit <= 0 {
		limit = 100000
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dynamicCacheTimeout)
	defer cancel()

	list, err := s.redis.ZRevRangeByScore(timeoutCtx, dynamicActiveUsersKey, &redis.ZRangeBy{
		Min:    strconv.FormatInt(since.Unix(), 10),
		Max:    "+inf",
		Offset: 0,
		Count:  limit,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (s *DynamicActiveStore) GetAuthorPolicy(ctx context.Context, authorUserID string) (DynamicAuthorPolicy, bool, error) {
	authorUserID = strings.TrimSpace(authorUserID)
	if s == nil || s.redis == nil || authorUserID == "" {
		return DynamicAuthorPolicy{}, false, nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dynamicCacheTimeout)
	defer cancel()

	content, err := s.redis.Get(timeoutCtx, dynamicAuthorPolicyKey(authorUserID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return DynamicAuthorPolicy{}, false, nil
	}
	if err != nil {
		return DynamicAuthorPolicy{}, false, err
	}

	var policy DynamicAuthorPolicy
	if err := json.Unmarshal(content, &policy); err != nil {
		return DynamicAuthorPolicy{}, false, err
	}
	return policy, true, nil
}

func (s *DynamicActiveStore) SetAuthorPolicy(ctx context.Context, policy DynamicAuthorPolicy) error {
	policy.AuthorUserID = strings.TrimSpace(policy.AuthorUserID)
	if s == nil || s.redis == nil || policy.AuthorUserID == "" {
		return nil
	}

	content, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dynamicCacheTimeout)
	defer cancel()

	return s.redis.Set(timeoutCtx, dynamicAuthorPolicyKey(policy.AuthorUserID), content, dynamicAuthorPolicyTTL).Err()
}

func (s *DynamicActiveStore) ListFeedItems(ctx context.Context, userID string, authorUserID string, cursor DynamicFeedCacheCursor, limit int) (DynamicFeedCachePage, error) {
	userID = strings.TrimSpace(userID)
	authorUserID = strings.TrimSpace(authorUserID)
	if s == nil || s.redis == nil || userID == "" {
		return DynamicFeedCachePage{}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dynamicCacheTimeout)
	defer cancel()

	key := dynamicFeedKey(userID, authorUserID)
	exists, err := s.redis.Exists(timeoutCtx, key).Result()
	if err != nil {
		return DynamicFeedCachePage{}, err
	}
	if exists == 0 {
		return DynamicFeedCachePage{Hit: false}, nil
	}

	maxScore := "+inf"
	if cursor.HasCursor {
		maxScore = strconv.FormatInt(cursor.Score, 10)
	}
	batchSize := int64(limit * 3)
	if batchSize < dynamicFeedScanBatchSize {
		batchSize = dynamicFeedScanBatchSize
	}

	items := make([]DynamicFeedCacheItem, 0, limit)
	var offset int64
	hasMore := false
	for len(items) < limit {
		zItems, err := s.redis.ZRevRangeByScoreWithScores(timeoutCtx, key, &redis.ZRangeBy{
			Min:    "-inf",
			Max:    maxScore,
			Offset: offset,
			Count:  batchSize,
		}).Result()
		if errors.Is(err, redis.Nil) {
			break
		}
		if err != nil {
			return DynamicFeedCachePage{}, err
		}
		if len(zItems) == 0 {
			break
		}
		offset += int64(len(zItems))
		for _, zItem := range zItems {
			member, ok := zItem.Member.(string)
			if !ok {
				continue
			}
			item := dynamicFeedCacheItemFromMember(member)
			score := int64(zItem.Score)
			if item.ContentID == "" || beforeOrAtDynamicCursor(score, item.ContentType, item.ContentID, cursor) {
				continue
			}
			item.AuthorUserID = authorUserID
			item.Score = score
			items = append(items, item)
			if len(items) >= limit {
				hasMore = true
				break
			}
		}
		if len(zItems) < int(batchSize) {
			break
		}
	}
	if items == nil {
		items = []DynamicFeedCacheItem{}
	}
	return DynamicFeedCachePage{Hit: true, Items: items, HasMore: hasMore}, nil
}

func (s *DynamicActiveStore) AddFeedItems(ctx context.Context, userID string, items []DynamicFeedCacheItem) error {
	userID = strings.TrimSpace(userID)
	if s == nil || s.redis == nil || userID == "" || len(items) == 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dynamicCacheTimeout)
	defer cancel()

	pipe := s.redis.Pipeline()
	addDynamicFeedItemsToPipe(timeoutCtx, pipe, userID, items)
	_, err := pipe.Exec(timeoutCtx)
	return err
}

func (s *DynamicActiveStore) AddFeedItemForUsers(ctx context.Context, userIDs []string, item DynamicFeedCacheItem) error {
	item = normalizeDynamicFeedCacheItem(item)
	item.AuthorUserID = strings.TrimSpace(item.AuthorUserID)
	if s == nil || s.redis == nil || item.ContentID == "" || item.Score <= 0 || len(userIDs) == 0 {
		return nil
	}

	for start := 0; start < len(userIDs); start += dynamicFeedWriteBatch {
		end := start + dynamicFeedWriteBatch
		if end > len(userIDs) {
			end = len(userIDs)
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, dynamicCacheTimeout)
		pipe := s.redis.Pipeline()
		for _, userID := range userIDs[start:end] {
			userID = strings.TrimSpace(userID)
			if userID == "" {
				continue
			}
			addDynamicFeedItemsToPipe(timeoutCtx, pipe, userID, []DynamicFeedCacheItem{item})
		}
		_, err := pipe.Exec(timeoutCtx)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DynamicActiveStore) RemoveFeedVideoIDs(ctx context.Context, userID string, authorUserID string, videoIDs []string) error {
	items := make([]DynamicFeedCacheItem, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		videoID = strings.TrimSpace(videoID)
		if videoID != "" {
			items = append(items, DynamicFeedCacheItem{VideoID: videoID})
		}
	}
	return s.RemoveFeedItems(ctx, userID, authorUserID, items)
}

func (s *DynamicActiveStore) RemoveFeedItems(ctx context.Context, userID string, authorUserID string, items []DynamicFeedCacheItem) error {
	userID = strings.TrimSpace(userID)
	authorUserID = strings.TrimSpace(authorUserID)
	if s == nil || s.redis == nil || userID == "" || len(items) == 0 {
		return nil
	}

	members := make([]any, 0, len(items))
	for _, item := range items {
		item = normalizeDynamicFeedCacheItem(item)
		if item.ContentID != "" {
			members = append(members, dynamicFeedCacheMember(item))
		}
	}
	if len(members) == 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dynamicCacheTimeout)
	defer cancel()

	pipe := s.redis.Pipeline()
	pipe.ZRem(timeoutCtx, dynamicFeedKey(userID, ""), members...)
	if authorUserID != "" {
		pipe.ZRem(timeoutCtx, dynamicFeedKey(userID, authorUserID), members...)
	}
	_, err := pipe.Exec(timeoutCtx)
	return err
}

func DynamicFeedScore(dynamicTime time.Time) int64 {
	return dynamicTime.Truncate(time.Second).UnixMilli()
}

func dynamicAuthorPolicyKey(authorUserID string) string {
	return "echo-space:dynamic:author-policy:" + authorUserID
}

func dynamicFeedKey(userID string, authorUserID string) string {
	key := "echo-space:dynamic:feed:" + userID
	if authorUserID != "" {
		key += ":author:" + authorUserID
	}
	return key
}

func addDynamicFeedItemsToPipe(ctx context.Context, pipe redis.Pipeliner, userID string, items []DynamicFeedCacheItem) {
	cutoffScore := dynamicFeedTrimCutoffScore(time.Now())
	for _, item := range items {
		item = normalizeDynamicFeedCacheItem(item)
		item.AuthorUserID = strings.TrimSpace(item.AuthorUserID)
		if item.ContentID == "" || item.Score <= 0 {
			continue
		}
		member := dynamicFeedCacheMember(item)
		fullKey := dynamicFeedKey(userID, "")
		pipe.ZAdd(ctx, fullKey, redis.Z{Score: float64(item.Score), Member: member})
		pipe.Expire(ctx, fullKey, dynamicFeedTTL)
		trimDynamicFeedKey(ctx, pipe, fullKey, cutoffScore)
		if item.AuthorUserID != "" {
			authorKey := dynamicFeedKey(userID, item.AuthorUserID)
			pipe.ZAdd(ctx, authorKey, redis.Z{Score: float64(item.Score), Member: member})
			pipe.Expire(ctx, authorKey, dynamicFeedTTL)
			trimDynamicFeedKey(ctx, pipe, authorKey, cutoffScore)
		}
	}
}

func trimDynamicFeedKey(ctx context.Context, pipe redis.Pipeliner, key string, cutoffScore int64) {
	pipe.ZRemRangeByRank(ctx, key, 0, dynamicFeedRankTrimStop())
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(cutoffScore, 10))
}

func dynamicFeedRankTrimStop() int64 {
	return -int64(dynamicFeedMaxCount) - 1
}

func dynamicFeedTrimCutoffScore(now time.Time) int64 {
	return DynamicFeedScore(now.Add(-dynamicFeedMaxAge))
}

func dynamicFeedCacheItemFromMember(member string) DynamicFeedCacheItem {
	member = strings.TrimSpace(member)
	if member == "" {
		return DynamicFeedCacheItem{}
	}
	if strings.HasPrefix(member, dynamicFeedImagePrefix) {
		contentID := strings.TrimSpace(strings.TrimPrefix(member, dynamicFeedImagePrefix))
		return DynamicFeedCacheItem{ContentType: 1, ContentID: contentID, VideoID: contentID}
	}
	return DynamicFeedCacheItem{ContentType: 0, ContentID: member, VideoID: member}
}

func normalizeDynamicFeedCacheItem(item DynamicFeedCacheItem) DynamicFeedCacheItem {
	item.VideoID = strings.TrimSpace(item.VideoID)
	item.ContentID = strings.TrimSpace(item.ContentID)
	if item.ContentID == "" {
		item.ContentID = item.VideoID
	}
	if item.VideoID == "" {
		item.VideoID = item.ContentID
	}
	if item.ContentType != 1 {
		item.ContentType = 0
	}
	return item
}

func dynamicFeedCacheMember(item DynamicFeedCacheItem) string {
	item = normalizeDynamicFeedCacheItem(item)
	if item.ContentType == 1 {
		return dynamicFeedImagePrefix + item.ContentID
	}
	return item.ContentID
}

func normalizeDynamicFeedCacheCursor(cursor DynamicFeedCacheCursor) DynamicFeedCacheCursor {
	cursor.ContentID = strings.TrimSpace(cursor.ContentID)
	cursor.VideoID = strings.TrimSpace(cursor.VideoID)
	if cursor.ContentID == "" {
		cursor.ContentID = cursor.VideoID
	}
	if cursor.VideoID == "" {
		cursor.VideoID = cursor.ContentID
	}
	if cursor.ContentType != 1 {
		cursor.ContentType = 0
	}
	return cursor
}

func beforeOrAtDynamicCursor(score int64, contentType int, contentID string, cursor DynamicFeedCacheCursor) bool {
	if !cursor.HasCursor {
		return false
	}
	cursor = normalizeDynamicFeedCacheCursor(cursor)
	contentID = strings.TrimSpace(contentID)
	if score > cursor.Score {
		return true
	}
	if score < cursor.Score {
		return false
	}
	if contentType > cursor.ContentType {
		return true
	}
	if contentType < cursor.ContentType {
		return false
	}
	return contentID >= cursor.ContentID
}
