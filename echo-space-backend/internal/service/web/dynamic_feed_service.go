package web

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

const (
	dynamicReadExpansionFanThreshold = 1000
	dynamicActiveUserWindow          = 7 * 24 * time.Hour
	dynamicActiveUserScanLimit       = 100000
	dynamicFeedOutboxScanInterval    = 5 * time.Second
	dynamicFeedOutboxBatchSize       = 20
	dynamicFeedPublishedRetryAge     = time.Minute
	dynamicFeedLeaseDuration         = time.Minute
	dynamicFeedMaxRetries            = 3
)

var dynamicFeedRetryDelays = [...]time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}

type DynamicFeedRepository interface {
	CountFans(ctx context.Context, authorUserID string) (int64, error)
	UpsertFeedForAllFollowers(ctx context.Context, message domain.DynamicFeedMessage) (int64, error)
	UpsertFeedForActiveFollowers(ctx context.Context, message domain.DynamicFeedMessage, activeUserIDs []string) (int64, error)
	ListFanoutUserIDs(ctx context.Context, authorUserID string, dynamicTime time.Time, candidateUserIDs []string) ([]string, error)
	ListDynamicFeedMessagesForPublish(ctx context.Context, limit int, publishedBefore time.Time) ([]domain.DynamicFeedMessageRecord, error)
	MarkDynamicFeedMessagePublished(ctx context.Context, messageID string, publishedBefore time.Time) error
	DelayDynamicFeedMessagePublish(ctx context.Context, messageID string, nextRetry time.Time, cause error) error
	ClaimDynamicFeedMessage(ctx context.Context, messageID string, lockToken string, leaseUntil time.Time) (*domain.DynamicFeedMessageRecord, bool, error)
	CompleteDynamicFeedMessage(ctx context.Context, messageID string, lockToken string) error
	RetryOrFailDynamicFeedMessage(ctx context.Context, messageID string, lockToken string, maxRetries int, nextRetry time.Time, cause error) error
}

type DynamicFeedPublisher interface {
	PublishDynamicFeedMessage(ctx context.Context, message domain.DynamicFeedMessage) error
}

type DynamicFeedActiveStore interface {
	ListActiveUserIDs(ctx context.Context, since time.Time, limit int64) ([]string, error)
	GetAuthorPolicy(ctx context.Context, authorUserID string) (cache.DynamicAuthorPolicy, bool, error)
	SetAuthorPolicy(ctx context.Context, policy cache.DynamicAuthorPolicy) error
}

type DynamicFeedFanoutCache interface {
	AddFeedItemForUsers(ctx context.Context, userIDs []string, item cache.DynamicFeedCacheItem) error
}

type DynamicFeedService struct {
	repository  DynamicFeedRepository
	activeStore DynamicFeedActiveStore
	feedCache   DynamicFeedFanoutCache
	publisher   DynamicFeedPublisher
	now         func() time.Time
}

func NewDynamicFeedService(repository DynamicFeedRepository, activeStore DynamicFeedActiveStore, publisher DynamicFeedPublisher, feedCaches ...DynamicFeedFanoutCache) *DynamicFeedService {
	service := &DynamicFeedService{
		repository: repository, activeStore: activeStore, publisher: publisher, now: time.Now,
	}
	if len(feedCaches) > 0 {
		service.feedCache = feedCaches[0]
	}
	return service
}

func (s *DynamicFeedService) StartOutboxPublisher(ctx context.Context) {
	go func() {
		s.publishPendingMessages(ctx)
		ticker := time.NewTicker(dynamicFeedOutboxScanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.publishPendingMessages(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *DynamicFeedService) HandleDynamicFeedMessage(ctx context.Context, incoming domain.DynamicFeedMessage) error {
	if s == nil || s.repository == nil {
		return errors.New("dynamic feed service is not ready")
	}

	lockToken, err := randomHexToken()
	if err != nil {
		return err
	}
	record, claimed, err := s.repository.ClaimDynamicFeedMessage(ctx, incoming.MessageID, lockToken, s.now().Add(dynamicFeedLeaseDuration))
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	var message domain.DynamicFeedMessage
	if err := json.Unmarshal([]byte(record.Payload), &message); err != nil {
		return s.persistDynamicFeedFailure(ctx, record.MessageID, lockToken, record.RetryCount, err)
	}
	if !sameDynamicFeedMessage(record, message) {
		return s.persistDynamicFeedFailure(ctx, record.MessageID, lockToken, record.RetryCount, errors.New("dynamic feed payload does not match outbox record"))
	}

	if err := s.spreadDynamicFeed(ctx, message); err != nil {
		return s.persistDynamicFeedFailure(ctx, message.MessageID, lockToken, record.RetryCount, err)
	}
	return s.repository.CompleteDynamicFeedMessage(ctx, message.MessageID, lockToken)
}

func (s *DynamicFeedService) publishPendingMessages(ctx context.Context) {
	if s == nil || s.repository == nil || s.publisher == nil {
		return
	}

	publishedBefore := s.now().Add(-dynamicFeedPublishedRetryAge)
	records, err := s.repository.ListDynamicFeedMessagesForPublish(ctx, dynamicFeedOutboxBatchSize, publishedBefore)
	if err != nil {
		log.Printf("list dynamic feed outbox failed: %v", err)
		return
	}
	for _, record := range records {
		var message domain.DynamicFeedMessage
		if err := json.Unmarshal([]byte(record.Payload), &message); err != nil {
			_ = s.repository.RetryOrFailDynamicFeedMessage(ctx, record.MessageID, "", 1, s.now(), err)
			continue
		}
		if err := s.publisher.PublishDynamicFeedMessage(ctx, message); err != nil {
			_ = s.repository.DelayDynamicFeedMessagePublish(ctx, record.MessageID, s.now().Add(dynamicFeedOutboxScanInterval), err)
			log.Printf("publish dynamic feed message failed, will retry: messageID=%s err=%v", record.MessageID, err)
			break
		}
		if err := s.repository.MarkDynamicFeedMessagePublished(ctx, record.MessageID, publishedBefore); err != nil {
			log.Printf("mark dynamic feed message published failed: messageID=%s err=%v", record.MessageID, err)
		}
	}
}

func (s *DynamicFeedService) spreadDynamicFeed(ctx context.Context, message domain.DynamicFeedMessage) error {
	policy, err := s.getAuthorPolicy(ctx, message.AuthorUserID)
	if err != nil {
		return err
	}
	if !policy.ReadExpansion {
		if _, err := s.repository.UpsertFeedForAllFollowers(ctx, message); err != nil {
			return err
		}
		s.cacheFanoutFeed(ctx, message, nil)
		return nil
	}

	activeUserIDs := []string{}
	if s.activeStore != nil {
		list, err := s.activeStore.ListActiveUserIDs(ctx, s.now().Add(-dynamicActiveUserWindow), dynamicActiveUserScanLimit)
		if err != nil {
			log.Printf("list dynamic active users failed, skip active pre-push: authorUserID=%s err=%v", message.AuthorUserID, err)
		} else {
			activeUserIDs = list
		}
	}
	if _, err = s.repository.UpsertFeedForActiveFollowers(ctx, message, activeUserIDs); err != nil {
		return err
	}
	s.cacheFanoutFeed(ctx, message, activeUserIDs)
	return nil
}

func (s *DynamicFeedService) cacheFanoutFeed(ctx context.Context, message domain.DynamicFeedMessage, candidateUserIDs []string) {
	if s == nil || s.repository == nil || s.feedCache == nil {
		return
	}
	userIDs, err := s.repository.ListFanoutUserIDs(ctx, message.AuthorUserID, message.DynamicTime, candidateUserIDs)
	if err != nil {
		log.Printf("list dynamic feed fanout users for cache failed: authorUserID=%s videoID=%s err=%v", message.AuthorUserID, message.VideoID, err)
		return
	}
	if len(userIDs) == 0 {
		return
	}
	err = s.feedCache.AddFeedItemForUsers(ctx, userIDs, cache.DynamicFeedCacheItem{
		ContentType:  dynamicMessageContentType(message),
		ContentID:    message.VideoID,
		VideoID:      message.VideoID,
		AuthorUserID: message.AuthorUserID,
		Score:        cache.DynamicFeedScore(message.DynamicTime),
	})
	if err != nil {
		log.Printf("write dynamic feed fanout cache failed: authorUserID=%s videoID=%s err=%v", message.AuthorUserID, message.VideoID, err)
	}
}

func dynamicMessageContentType(message domain.DynamicFeedMessage) int {
	if message.EventType == domain.DynamicEventTypeImage {
		return domain.ContentTypeImage
	}
	return domain.ContentTypeVideo
}

func (s *DynamicFeedService) getAuthorPolicy(ctx context.Context, authorUserID string) (cache.DynamicAuthorPolicy, error) {
	if s.activeStore != nil {
		policy, ok, err := s.activeStore.GetAuthorPolicy(ctx, authorUserID)
		if err == nil && ok {
			return policy, nil
		}
		if err != nil {
			log.Printf("get dynamic author policy cache failed: authorUserID=%s err=%v", authorUserID, err)
		}
	}

	fansCount, err := s.repository.CountFans(ctx, authorUserID)
	if err != nil {
		return cache.DynamicAuthorPolicy{}, err
	}
	policy := cache.DynamicAuthorPolicy{
		AuthorUserID:    authorUserID,
		FansCount:       fansCount,
		ReadExpansion:   fansCount >= dynamicReadExpansionFanThreshold,
		CalculatedAtSec: s.now().Unix(),
	}
	if s.activeStore != nil {
		if err := s.activeStore.SetAuthorPolicy(ctx, policy); err != nil {
			log.Printf("set dynamic author policy cache failed: authorUserID=%s err=%v", authorUserID, err)
		}
	}
	return policy, nil
}

func (s *DynamicFeedService) persistDynamicFeedFailure(ctx context.Context, messageID string, lockToken string, retryCount int, cause error) error {
	delayIndex := retryCount
	if delayIndex >= len(dynamicFeedRetryDelays) {
		delayIndex = len(dynamicFeedRetryDelays) - 1
	}
	if err := s.repository.RetryOrFailDynamicFeedMessage(ctx, messageID, lockToken, dynamicFeedMaxRetries, s.now().Add(dynamicFeedRetryDelays[delayIndex]), cause); err != nil {
		return err
	}
	log.Printf("dynamic feed task scheduled for retry: messageID=%s err=%v", messageID, cause)
	return nil
}

func sameDynamicFeedMessage(record *domain.DynamicFeedMessageRecord, message domain.DynamicFeedMessage) bool {
	return record != nil &&
		record.MessageID == message.MessageID &&
		record.EventID == message.EventID &&
		record.VideoID == message.VideoID &&
		record.AuthorUserID == message.AuthorUserID
}
