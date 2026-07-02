package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultDynamicPageSize = 10
	maxDynamicPageSize     = 30
	dynamicTimeLayout      = "2006-01-02 15:04:05"
	dynamicCacheReadLimit  = 5
)

type DynamicRepository interface {
	FindCurrentUserInfo(ctx context.Context, userID string) (*domain.DynamicCurrentUserInfo, error)
	ListFollowUsers(ctx context.Context, userID string) ([]domain.DynamicFollowUserItem, error)
	ListFeedByCursor(ctx context.Context, query repository.DynamicFeedQuery) ([]domain.WebVideoItem, error)
	ListFeedContentDetailsByKeys(ctx context.Context, keys []domain.DynamicFeedContentKey) ([]domain.WebVideoItem, error)
	UpsertFeedItems(ctx context.Context, userID string, list []domain.WebVideoItem) error
}

type DynamicFeedCache interface {
	ListFeedItems(ctx context.Context, userID string, authorUserID string, cursor cache.DynamicFeedCacheCursor, limit int) (cache.DynamicFeedCachePage, error)
	AddFeedItems(ctx context.Context, userID string, items []cache.DynamicFeedCacheItem) error
	RemoveFeedItems(ctx context.Context, userID string, authorUserID string, items []cache.DynamicFeedCacheItem) error
}

type DynamicService struct {
	repository DynamicRepository
	feedCache  DynamicFeedCache
}

type LoadDynamicFeedInput struct {
	UserID      string
	FocusUserID string
	Cursor      string
	PageSize    int
}

type dynamicCursorPayload struct {
	FocusUserID     string `json:"focusUserId,omitempty"`
	LastUpdateTime  string `json:"lastUpdateTime"`
	LastContentType int    `json:"lastContentType,omitempty"`
	LastContentID   string `json:"lastContentId,omitempty"`
	LastVideoID     string `json:"lastVideoId,omitempty"`
}

func NewDynamicService(repository DynamicRepository, feedCaches ...DynamicFeedCache) *DynamicService {
	service := &DynamicService{repository: repository}
	if len(feedCaches) > 0 {
		service.feedCache = feedCaches[0]
	}
	return service
}

func (s *DynamicService) LoadCurrentUserInfo(ctx context.Context, userID string) (*domain.DynamicCurrentUserInfo, error) {
	userID = strings.TrimSpace(userID)
	if !validWebUserID(userID) {
		return nil, &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("dynamic service is not ready")
	}

	info, err := s.repository.FindCurrentUserInfo(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "用户不存在"}
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (s *DynamicService) LoadFollowUsers(ctx context.Context, userID string) ([]domain.DynamicFollowUserItem, error) {
	userID = strings.TrimSpace(userID)
	if !validWebUserID(userID) {
		return nil, &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("dynamic service is not ready")
	}

	list, err := s.repository.ListFollowUsers(ctx, userID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.DynamicFollowUserItem{}
	}
	return list, nil
}

func (s *DynamicService) LoadFeed(ctx context.Context, input LoadDynamicFeedInput) (domain.DynamicFeedPage, error) {
	input = normalizeDynamicFeedInput(input)
	if err := validateDynamicFeedInput(input); err != nil {
		return domain.DynamicFeedPage{}, err
	}
	if s == nil || s.repository == nil {
		return domain.DynamicFeedPage{}, errors.New("dynamic service is not ready")
	}

	cursor, err := decodeDynamicCursor(input)
	if err != nil {
		return domain.DynamicFeedPage{}, err
	}

	if s.feedCache != nil {
		page, ok, err := s.loadFeedFromCache(ctx, input, cursor)
		if err == nil && ok {
			return page, nil
		}
		if err != nil {
			log.Printf("load dynamic feed from redis failed, fallback to database: userID=%s err=%v", input.UserID, err)
		}
	}

	return s.loadFeedFromDatabase(ctx, input, cursor)
}

func (s *DynamicService) loadFeedFromDatabase(ctx context.Context, input LoadDynamicFeedInput, cursor dynamicCursorPayload) (domain.DynamicFeedPage, error) {
	list, err := s.repository.ListFeedByCursor(ctx, repository.DynamicFeedQuery{
		UserID:          input.UserID,
		FocusUserID:     input.FocusUserID,
		PageSize:        input.PageSize + 1,
		LastUpdateTime:  cursor.LastUpdateTime,
		LastContentType: cursor.LastContentType,
		LastContentID:   cursor.LastContentID,
		LastVideoID:     cursor.LastVideoID,
		ReadFanCount:    dynamicReadExpansionFanThreshold,
	})
	if err != nil {
		return domain.DynamicFeedPage{}, err
	}

	fullList := list
	hasMore := len(list) > input.PageSize
	if hasMore {
		list = list[:input.PageSize]
	}
	if list == nil {
		list = []domain.WebVideoItem{}
	}
	fillWebVideoPlayTime(list)
	if err := s.repository.UpsertFeedItems(ctx, input.UserID, fullList); err != nil {
		log.Printf("lazy upsert dynamic feed items failed: userID=%s err=%v", input.UserID, err)
	}
	s.cacheDynamicFeedItems(ctx, input.UserID, fullList)

	nextCursor := ""
	if hasMore && len(list) > 0 {
		last := list[len(list)-1]
		nextCursor, err = encodeDynamicCursor(dynamicCursorPayload{
			FocusUserID:     input.FocusUserID,
			LastUpdateTime:  last.LastUpdateTime,
			LastContentType: normalizeDynamicContentType(last.ContentType),
			LastContentID:   dynamicFeedItemContentID(last),
			LastVideoID:     dynamicFeedItemContentID(last),
		})
		if err != nil {
			return domain.DynamicFeedPage{}, err
		}
	}

	return domain.DynamicFeedPage{
		PageSize:   input.PageSize,
		List:       list,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *DynamicService) loadFeedFromCache(ctx context.Context, input LoadDynamicFeedInput, cursor dynamicCursorPayload) (domain.DynamicFeedPage, bool, error) {
	cacheCursor, err := dynamicCacheCursorFromPayload(cursor)
	if err != nil {
		return domain.DynamicFeedPage{}, false, err
	}

	targetSize := input.PageSize + 1
	list := make([]domain.WebVideoItem, 0, targetSize)
	currentCursor := cacheCursor
	cacheHit := false
	var dirtyItems []cache.DynamicFeedCacheItem

	for i := 0; i < dynamicCacheReadLimit && len(list) < targetSize; i++ {
		cachePage, err := s.feedCache.ListFeedItems(ctx, input.UserID, input.FocusUserID, currentCursor, targetSize-len(list))
		if err != nil {
			return domain.DynamicFeedPage{}, false, err
		}
		if !cachePage.Hit {
			return domain.DynamicFeedPage{}, false, nil
		}
		cacheHit = true
		if len(cachePage.Items) == 0 {
			break
		}

		keys := dynamicFeedCacheContentKeys(cachePage.Items)
		details, err := s.repository.ListFeedContentDetailsByKeys(ctx, keys)
		if err != nil {
			return domain.DynamicFeedPage{}, false, err
		}
		orderedList, missingItems := orderDynamicFeedDetailsByCacheItems(cachePage.Items, details)
		list = append(list, orderedList...)
		dirtyItems = append(dirtyItems, missingItems...)

		lastCacheItem := cachePage.Items[len(cachePage.Items)-1]
		currentCursor = cache.DynamicFeedCacheCursor{
			HasCursor:   true,
			Score:       lastCacheItem.Score,
			ContentType: normalizeDynamicContentType(lastCacheItem.ContentType),
			ContentID:   dynamicFeedCacheItemContentID(lastCacheItem),
			VideoID:     dynamicFeedCacheItemContentID(lastCacheItem),
		}
		if !cachePage.HasMore {
			break
		}
	}
	if len(dirtyItems) > 0 {
		if err := s.feedCache.RemoveFeedItems(ctx, input.UserID, input.FocusUserID, dirtyItems); err != nil {
			log.Printf("remove dirty dynamic feed cache items failed: userID=%s err=%v", input.UserID, err)
		}
	}
	if !cacheHit {
		return domain.DynamicFeedPage{}, false, nil
	}
	if cursor.LastUpdateTime != "" && len(list) < targetSize {
		return domain.DynamicFeedPage{}, false, nil
	}
	if list == nil {
		list = []domain.WebVideoItem{}
	}
	fillWebVideoPlayTime(list)
	return buildDynamicFeedPage(input, list)
}

func buildDynamicFeedPage(input LoadDynamicFeedInput, list []domain.WebVideoItem) (domain.DynamicFeedPage, bool, error) {
	hasMore := len(list) > input.PageSize
	if hasMore {
		list = list[:input.PageSize]
	}
	if list == nil {
		list = []domain.WebVideoItem{}
	}

	nextCursor := ""
	if hasMore && len(list) > 0 {
		last := list[len(list)-1]
		cursor, err := encodeDynamicCursor(dynamicCursorPayload{
			FocusUserID:     input.FocusUserID,
			LastUpdateTime:  last.LastUpdateTime,
			LastContentType: normalizeDynamicContentType(last.ContentType),
			LastContentID:   dynamicFeedItemContentID(last),
			LastVideoID:     dynamicFeedItemContentID(last),
		})
		if err != nil {
			return domain.DynamicFeedPage{}, false, err
		}
		nextCursor = cursor
	}

	return domain.DynamicFeedPage{
		PageSize:   input.PageSize,
		List:       list,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, true, nil
}

func (s *DynamicService) cacheDynamicFeedItems(ctx context.Context, userID string, list []domain.WebVideoItem) {
	if s == nil || s.feedCache == nil || len(list) == 0 {
		return
	}
	items := dynamicFeedCacheItemsFromVideos(list)
	if len(items) == 0 {
		return
	}
	if err := s.feedCache.AddFeedItems(ctx, userID, items); err != nil {
		log.Printf("write dynamic feed cache failed: userID=%s err=%v", userID, err)
	}
}

func dynamicFeedCacheItemsFromVideos(list []domain.WebVideoItem) []cache.DynamicFeedCacheItem {
	items := make([]cache.DynamicFeedCacheItem, 0, len(list))
	for _, item := range list {
		dynamicTime, err := time.ParseInLocation(dynamicTimeLayout, item.LastUpdateTime, time.Local)
		contentID := dynamicFeedItemContentID(item)
		if err != nil || contentID == "" {
			continue
		}
		items = append(items, cache.DynamicFeedCacheItem{
			ContentType:  normalizeDynamicContentType(item.ContentType),
			ContentID:    contentID,
			VideoID:      item.VideoID,
			AuthorUserID: item.UserID,
			Score:        cache.DynamicFeedScore(dynamicTime),
		})
	}
	return items
}

func dynamicCacheCursorFromPayload(payload dynamicCursorPayload) (cache.DynamicFeedCacheCursor, error) {
	if strings.TrimSpace(payload.LastUpdateTime) == "" {
		return cache.DynamicFeedCacheCursor{}, nil
	}
	dynamicTime, err := time.ParseInLocation(dynamicTimeLayout, payload.LastUpdateTime, time.Local)
	if err != nil {
		return cache.DynamicFeedCacheCursor{}, &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	return cache.DynamicFeedCacheCursor{
		HasCursor:   true,
		Score:       cache.DynamicFeedScore(dynamicTime),
		ContentType: normalizeDynamicContentType(payload.LastContentType),
		ContentID:   payload.LastContentID,
		VideoID:     payload.LastContentID,
	}, nil
}

func dynamicFeedCacheContentKeys(items []cache.DynamicFeedCacheItem) []domain.DynamicFeedContentKey {
	keys := make([]domain.DynamicFeedContentKey, 0, len(items))
	for _, item := range items {
		contentID := dynamicFeedCacheItemContentID(item)
		if contentID != "" {
			keys = append(keys, domain.DynamicFeedContentKey{
				ContentType: normalizeDynamicContentType(item.ContentType),
				ContentID:   contentID,
			})
		}
	}
	return keys
}

func orderDynamicFeedDetailsByCacheItems(items []cache.DynamicFeedCacheItem, details []domain.WebVideoItem) ([]domain.WebVideoItem, []cache.DynamicFeedCacheItem) {
	detailMap := make(map[string]domain.WebVideoItem, len(details))
	for _, detail := range details {
		detailMap[dynamicFeedItemKey(normalizeDynamicContentType(detail.ContentType), dynamicFeedItemContentID(detail))] = detail
	}

	orderedList := make([]domain.WebVideoItem, 0, len(items))
	missingItems := make([]cache.DynamicFeedCacheItem, 0)
	for _, item := range items {
		contentType := normalizeDynamicContentType(item.ContentType)
		contentID := dynamicFeedCacheItemContentID(item)
		detail, ok := detailMap[dynamicFeedItemKey(contentType, contentID)]
		if !ok {
			missingItems = append(missingItems, cache.DynamicFeedCacheItem{ContentType: contentType, ContentID: contentID, VideoID: contentID})
			continue
		}
		orderedList = append(orderedList, detail)
	}
	return orderedList, missingItems
}

func normalizeDynamicFeedInput(input LoadDynamicFeedInput) LoadDynamicFeedInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.FocusUserID = strings.TrimSpace(input.FocusUserID)
	input.Cursor = strings.TrimSpace(input.Cursor)
	if input.PageSize <= 0 {
		input.PageSize = defaultDynamicPageSize
	}
	if input.PageSize > maxDynamicPageSize {
		input.PageSize = maxDynamicPageSize
	}
	return input
}

func validateDynamicFeedInput(input LoadDynamicFeedInput) error {
	if !validWebUserID(input.UserID) {
		return &BusinessError{Info: "参数错误"}
	}
	if input.FocusUserID != "" && !validWebUserID(input.FocusUserID) {
		return &BusinessError{Info: "参数错误"}
	}
	return nil
}

func decodeDynamicCursor(input LoadDynamicFeedInput) (dynamicCursorPayload, error) {
	if input.Cursor == "" {
		return dynamicCursorPayload{}, nil
	}

	content, err := base64.RawURLEncoding.DecodeString(input.Cursor)
	if err != nil {
		return dynamicCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	var payload dynamicCursorPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return dynamicCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	payload.LastContentID = strings.TrimSpace(payload.LastContentID)
	payload.LastVideoID = strings.TrimSpace(payload.LastVideoID)
	if payload.LastContentID == "" {
		payload.LastContentID = payload.LastVideoID
	}
	if payload.LastVideoID == "" {
		payload.LastVideoID = payload.LastContentID
	}
	payload.LastContentType = normalizeDynamicContentType(payload.LastContentType)
	if payload.FocusUserID != input.FocusUserID ||
		strings.TrimSpace(payload.LastUpdateTime) == "" ||
		len(payload.LastContentID) != 10 ||
		!isValidPublicVideoID(payload.LastContentID) {
		return dynamicCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	return payload, nil
}

func encodeDynamicCursor(payload dynamicCursorPayload) (string, error) {
	content, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(content), nil
}

func dynamicFeedItemContentID(item domain.WebVideoItem) string {
	contentID := strings.TrimSpace(item.ContentID)
	if contentID != "" {
		return contentID
	}
	return strings.TrimSpace(item.VideoID)
}

func dynamicFeedCacheItemContentID(item cache.DynamicFeedCacheItem) string {
	contentID := strings.TrimSpace(item.ContentID)
	if contentID != "" {
		return contentID
	}
	return strings.TrimSpace(item.VideoID)
}

func normalizeDynamicContentType(contentType int) int {
	if contentType == domain.ContentTypeImage {
		return domain.ContentTypeImage
	}
	return domain.ContentTypeVideo
}

func dynamicFeedItemKey(contentType int, contentID string) string {
	if normalizeDynamicContentType(contentType) == domain.ContentTypeImage {
		return "1:" + strings.TrimSpace(contentID)
	}
	return "0:" + strings.TrimSpace(contentID)
}
