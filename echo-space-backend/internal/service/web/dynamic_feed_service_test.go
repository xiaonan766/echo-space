package web

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

func TestDynamicFeedSmallAuthorWritesAllFollowers(t *testing.T) {
	repo := &fakeDynamicFeedRepository{fansCount: dynamicReadExpansionFanThreshold - 1}
	service := NewDynamicFeedService(repo, &fakeDynamicActiveStore{}, nil)
	service.now = fixedDynamicNow

	err := service.spreadDynamicFeed(context.Background(), testDynamicFeedMessage())
	if err != nil {
		t.Fatalf("spread dynamic feed returned error: %v", err)
	}
	if repo.allFollowerWrites != 1 {
		t.Fatalf("all follower writes = %d, want 1", repo.allFollowerWrites)
	}
	if repo.activeFollowerWrites != 0 {
		t.Fatalf("active follower writes = %d, want 0", repo.activeFollowerWrites)
	}
}

func TestDynamicFeedBigAuthorPrePushesActiveFollowers(t *testing.T) {
	activeStore := &fakeDynamicActiveStore{activeUserIDs: []string{"1000000001", "1000000002"}}
	repo := &fakeDynamicFeedRepository{fansCount: dynamicReadExpansionFanThreshold}
	service := NewDynamicFeedService(repo, activeStore, nil)
	service.now = fixedDynamicNow

	err := service.spreadDynamicFeed(context.Background(), testDynamicFeedMessage())
	if err != nil {
		t.Fatalf("spread dynamic feed returned error: %v", err)
	}
	if repo.allFollowerWrites != 0 {
		t.Fatalf("all follower writes = %d, want 0", repo.allFollowerWrites)
	}
	if repo.activeFollowerWrites != 1 {
		t.Fatalf("active follower writes = %d, want 1", repo.activeFollowerWrites)
	}
	if len(repo.activeUserIDs) != 2 {
		t.Fatalf("active user ids = %#v, want two ids", repo.activeUserIDs)
	}
}

func TestDynamicFeedWritesRedisZSetAfterFanout(t *testing.T) {
	feedCache := &fakeDynamicFanoutCache{}
	repo := &fakeDynamicFeedRepository{
		fansCount:     dynamicReadExpansionFanThreshold - 1,
		fanoutUserIDs: []string{"1000000001", "1000000002"},
	}
	service := NewDynamicFeedService(repo, &fakeDynamicActiveStore{}, nil, feedCache)
	service.now = fixedDynamicNow

	err := service.spreadDynamicFeed(context.Background(), testDynamicFeedMessage())
	if err != nil {
		t.Fatalf("spread dynamic feed returned error: %v", err)
	}
	if len(feedCache.userIDs) != 2 {
		t.Fatalf("cache user ids = %#v, want two users", feedCache.userIDs)
	}
	if feedCache.item.VideoID != "Video00001" || feedCache.item.AuthorUserID != "2000000001" {
		t.Fatalf("cache item = %#v, want dynamic video item", feedCache.item)
	}
}

func TestDynamicFeedImageFanoutKeepsEventType(t *testing.T) {
	feedCache := &fakeDynamicFanoutCache{}
	repo := &fakeDynamicFeedRepository{
		fansCount:     dynamicReadExpansionFanThreshold - 1,
		fanoutUserIDs: []string{"1000000001"},
	}
	service := NewDynamicFeedService(repo, &fakeDynamicActiveStore{}, nil, feedCache)
	service.now = fixedDynamicNow
	message := testDynamicFeedMessage()
	message.EventID = "image_Image00001"
	message.VideoID = "Image00001"
	message.EventType = domain.DynamicEventTypeImage

	err := service.spreadDynamicFeed(context.Background(), message)
	if err != nil {
		t.Fatalf("spread dynamic feed returned error: %v", err)
	}
	if repo.lastAllFollowerMessage.EventType != domain.DynamicEventTypeImage {
		t.Fatalf("fanout event type = %d, want image", repo.lastAllFollowerMessage.EventType)
	}
	if feedCache.item.ContentType != domain.ContentTypeImage || feedCache.item.ContentID != "Image00001" {
		t.Fatalf("cache item = %#v, want image content item", feedCache.item)
	}
}

func TestDynamicFeedUsesCachedAuthorPolicy(t *testing.T) {
	activeStore := &fakeDynamicActiveStore{
		cachedPolicy: cache.DynamicAuthorPolicy{
			AuthorUserID:  "2000000001",
			FansCount:     dynamicReadExpansionFanThreshold,
			ReadExpansion: true,
		},
		hasCachedPolicy: true,
		activeUserIDs:   []string{"1000000001"},
	}
	repo := &fakeDynamicFeedRepository{fansCount: 0}
	service := NewDynamicFeedService(repo, activeStore, nil)
	service.now = fixedDynamicNow

	err := service.spreadDynamicFeed(context.Background(), testDynamicFeedMessage())
	if err != nil {
		t.Fatalf("spread dynamic feed returned error: %v", err)
	}
	if repo.countFansCalls != 0 {
		t.Fatalf("count fans calls = %d, want 0", repo.countFansCalls)
	}
	if repo.activeFollowerWrites != 1 {
		t.Fatalf("active follower writes = %d, want 1", repo.activeFollowerWrites)
	}
}

func TestDynamicFeedOutboxKeepsMessageWhenPublishFails(t *testing.T) {
	message := testDynamicFeedMessage()
	payload := `{"messageId":"` + message.MessageID + `","eventId":"` + message.EventID + `","videoId":"` + message.VideoID + `","authorUserId":"` + message.AuthorUserID + `","dynamicTime":"2026-06-29T10:00:00+08:00","eventType":1}`
	repo := &fakeDynamicFeedRepository{
		pendingMessages: []domain.DynamicFeedMessageRecord{{
			MessageID: message.MessageID, EventID: message.EventID, VideoID: message.VideoID, AuthorUserID: message.AuthorUserID, Payload: payload,
		}},
	}
	service := NewDynamicFeedService(repo, nil, &fakeDynamicFeedPublisher{err: errors.New("rabbitmq down")})
	service.now = fixedDynamicNow

	service.publishPendingMessages(context.Background())
	if repo.delayPublishCalls != 1 {
		t.Fatalf("delay publish calls = %d, want 1", repo.delayPublishCalls)
	}
	if repo.markPublishedCalls != 0 {
		t.Fatalf("mark published calls = %d, want 0", repo.markPublishedCalls)
	}
}

func fixedDynamicNow() time.Time {
	return time.Date(2026, 6, 29, 10, 0, 0, 0, time.Local)
}

func testDynamicFeedMessage() domain.DynamicFeedMessage {
	return domain.DynamicFeedMessage{
		MessageID:    "df-test-message",
		EventID:      "video_Video00001",
		VideoID:      "Video00001",
		AuthorUserID: "2000000001",
		DynamicTime:  fixedDynamicNow(),
		EventType:    domain.DynamicEventTypeVideo,
	}
}

type fakeDynamicFeedRepository struct {
	fansCount int64

	countFansCalls            int
	allFollowerWrites         int
	activeFollowerWrites      int
	activeUserIDs             []string
	fanoutUserIDs             []string
	lastAllFollowerMessage    domain.DynamicFeedMessage
	lastActiveFollowerMessage domain.DynamicFeedMessage

	pendingMessages    []domain.DynamicFeedMessageRecord
	markPublishedCalls int
	delayPublishCalls  int
}

func (r *fakeDynamicFeedRepository) CountFans(ctx context.Context, authorUserID string) (int64, error) {
	r.countFansCalls++
	return r.fansCount, nil
}

func (r *fakeDynamicFeedRepository) UpsertFeedForAllFollowers(ctx context.Context, message domain.DynamicFeedMessage) (int64, error) {
	r.allFollowerWrites++
	r.lastAllFollowerMessage = message
	return 1, nil
}

func (r *fakeDynamicFeedRepository) UpsertFeedForActiveFollowers(ctx context.Context, message domain.DynamicFeedMessage, activeUserIDs []string) (int64, error) {
	r.activeFollowerWrites++
	r.lastActiveFollowerMessage = message
	r.activeUserIDs = append([]string(nil), activeUserIDs...)
	return int64(len(activeUserIDs)), nil
}

func (r *fakeDynamicFeedRepository) ListFanoutUserIDs(ctx context.Context, authorUserID string, dynamicTime time.Time, candidateUserIDs []string) ([]string, error) {
	if r.fanoutUserIDs != nil {
		return r.fanoutUserIDs, nil
	}
	return candidateUserIDs, nil
}

func (r *fakeDynamicFeedRepository) ListDynamicFeedMessagesForPublish(ctx context.Context, limit int, publishedBefore time.Time) ([]domain.DynamicFeedMessageRecord, error) {
	return r.pendingMessages, nil
}

func (r *fakeDynamicFeedRepository) MarkDynamicFeedMessagePublished(ctx context.Context, messageID string, publishedBefore time.Time) error {
	r.markPublishedCalls++
	return nil
}

func (r *fakeDynamicFeedRepository) DelayDynamicFeedMessagePublish(ctx context.Context, messageID string, nextRetry time.Time, cause error) error {
	r.delayPublishCalls++
	return nil
}

func (r *fakeDynamicFeedRepository) ClaimDynamicFeedMessage(ctx context.Context, messageID string, lockToken string, leaseUntil time.Time) (*domain.DynamicFeedMessageRecord, bool, error) {
	return nil, false, nil
}

func (r *fakeDynamicFeedRepository) CompleteDynamicFeedMessage(ctx context.Context, messageID string, lockToken string) error {
	return nil
}

func (r *fakeDynamicFeedRepository) RetryOrFailDynamicFeedMessage(ctx context.Context, messageID string, lockToken string, maxRetries int, nextRetry time.Time, cause error) error {
	return nil
}

type fakeDynamicActiveStore struct {
	activeUserIDs   []string
	cachedPolicy    cache.DynamicAuthorPolicy
	hasCachedPolicy bool
}

func (s *fakeDynamicActiveStore) ListActiveUserIDs(ctx context.Context, since time.Time, limit int64) ([]string, error) {
	return s.activeUserIDs, nil
}

func (s *fakeDynamicActiveStore) GetAuthorPolicy(ctx context.Context, authorUserID string) (cache.DynamicAuthorPolicy, bool, error) {
	return s.cachedPolicy, s.hasCachedPolicy, nil
}

func (s *fakeDynamicActiveStore) SetAuthorPolicy(ctx context.Context, policy cache.DynamicAuthorPolicy) error {
	s.cachedPolicy = policy
	s.hasCachedPolicy = true
	return nil
}

type fakeDynamicFeedPublisher struct {
	err error
}

func (p *fakeDynamicFeedPublisher) PublishDynamicFeedMessage(ctx context.Context, message domain.DynamicFeedMessage) error {
	return p.err
}

type fakeDynamicFanoutCache struct {
	userIDs []string
	item    cache.DynamicFeedCacheItem
}

func (c *fakeDynamicFanoutCache) AddFeedItemForUsers(ctx context.Context, userIDs []string, item cache.DynamicFeedCacheItem) error {
	c.userIDs = append([]string(nil), userIDs...)
	c.item = item
	return nil
}
