package web

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

func TestLoadDynamicFollowUsersReturnsEmptyList(t *testing.T) {
	repo := &fakeDynamicRepository{}
	service := NewDynamicService(repo)

	list, err := service.LoadFollowUsers(context.Background(), "1000000001")
	if err != nil {
		t.Fatalf("load follow users returned error: %v", err)
	}
	if list == nil {
		t.Fatal("follow user list is nil, want empty slice")
	}
	if len(list) != 0 {
		t.Fatalf("follow user list len = %d, want 0", len(list))
	}
}

func TestLoadDynamicCurrentUserInfo(t *testing.T) {
	repo := &fakeDynamicRepository{
		currentUserInfo: &domain.DynamicCurrentUserInfo{
			UserID:       "1000000001",
			NickName:     "tester",
			FocusCount:   3,
			FansCount:    2,
			DynamicCount: 5,
		},
	}
	service := NewDynamicService(repo)

	result, err := service.LoadCurrentUserInfo(context.Background(), "1000000001")
	if err != nil {
		t.Fatalf("load current user info returned error: %v", err)
	}
	if result.DynamicCount != 5 {
		t.Fatalf("dynamic count = %d, want 5", result.DynamicCount)
	}
}

func TestLoadDynamicFeedBuildsNextCursor(t *testing.T) {
	repo := &fakeDynamicRepository{
		feed: []domain.WebVideoItem{
			{VideoID: "Video00003", LastUpdateTime: "2026-06-25 10:03:00", Duration: 65},
			{VideoID: "Video00002", LastUpdateTime: "2026-06-25 10:02:00", Duration: 61},
			{VideoID: "Video00001", LastUpdateTime: "2026-06-25 10:01:00", Duration: 59},
		},
	}
	service := NewDynamicService(repo)

	result, err := service.LoadFeed(context.Background(), LoadDynamicFeedInput{
		UserID:   "1000000001",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("load feed returned error: %v", err)
	}
	if repo.lastQuery.PageSize != 3 {
		t.Fatalf("query page size = %d, want 3", repo.lastQuery.PageSize)
	}
	if len(result.List) != 2 {
		t.Fatalf("feed len = %d, want 2", len(result.List))
	}
	if !result.HasMore {
		t.Fatal("hasMore = false, want true")
	}
	if result.List[1].PlayTime != "01:01" {
		t.Fatalf("playTime = %s, want 01:01", result.List[1].PlayTime)
	}

	cursor, err := decodeDynamicCursor(LoadDynamicFeedInput{Cursor: result.NextCursor})
	if err != nil {
		t.Fatalf("decode next cursor: %v", err)
	}
	wantCursor := dynamicCursorPayload{
		LastUpdateTime: "2026-06-25 10:02:00",
		LastVideoID:    "Video00002",
	}
	if !reflect.DeepEqual(cursor, wantCursor) {
		t.Fatalf("cursor = %#v, want %#v", cursor, wantCursor)
	}
}

func TestLoadDynamicFeedUsesCursorAndFocusUser(t *testing.T) {
	cursor, err := encodeDynamicCursor(dynamicCursorPayload{
		FocusUserID:    "1000000002",
		LastUpdateTime: "2026-06-25 10:02:00",
		LastVideoID:    "Video00002",
	})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	repo := &fakeDynamicRepository{}
	service := NewDynamicService(repo)

	_, err = service.LoadFeed(context.Background(), LoadDynamicFeedInput{
		UserID:      "1000000001",
		FocusUserID: "1000000002",
		Cursor:      cursor,
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("load feed returned error: %v", err)
	}
	if repo.lastQuery.FocusUserID != "1000000002" {
		t.Fatalf("focus user id = %s, want 1000000002", repo.lastQuery.FocusUserID)
	}
	if repo.lastQuery.LastUpdateTime != "2026-06-25 10:02:00" || repo.lastQuery.LastVideoID != "Video00002" {
		t.Fatalf("cursor query = %#v, want last update/video cursor", repo.lastQuery)
	}
}

func TestLoadDynamicFeedRejectsCursorMismatch(t *testing.T) {
	cursor, err := encodeDynamicCursor(dynamicCursorPayload{
		FocusUserID:    "1000000002",
		LastUpdateTime: "2026-06-25 10:02:00",
		LastVideoID:    "Video00002",
	})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	service := NewDynamicService(&fakeDynamicRepository{})

	_, err = service.LoadFeed(context.Background(), LoadDynamicFeedInput{
		UserID:      "1000000001",
		FocusUserID: "1000000003",
		Cursor:      cursor,
	})
	if err == nil {
		t.Fatal("expected cursor mismatch error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %T %v, want BusinessError", err, err)
	}
}

func TestLoadDynamicFeedUsesRedisZSetCache(t *testing.T) {
	repo := &fakeDynamicRepository{
		detailByID: map[string]domain.WebVideoItem{
			"Video00003": {VideoID: "Video00003", UserID: "2000000001", LastUpdateTime: "2026-06-25 10:03:00", Duration: 65},
			"Video00002": {VideoID: "Video00002", UserID: "2000000001", LastUpdateTime: "2026-06-25 10:02:00", Duration: 61},
		},
	}
	feedCache := &fakeDynamicFeedCache{
		page: cache.DynamicFeedCachePage{
			Hit: true,
			Items: []cache.DynamicFeedCacheItem{
				{VideoID: "Video00003", Score: 1782352980000},
				{VideoID: "Video00002", Score: 1782352920000},
			},
		},
	}
	service := NewDynamicService(repo, feedCache)

	result, err := service.LoadFeed(context.Background(), LoadDynamicFeedInput{
		UserID:   "1000000001",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("load feed returned error: %v", err)
	}
	if repo.listFeedCalls != 0 {
		t.Fatalf("database feed calls = %d, want 0", repo.listFeedCalls)
	}
	if len(result.List) != 2 || result.List[0].VideoID != "Video00003" || result.List[1].VideoID != "Video00002" {
		t.Fatalf("cache feed list = %#v, want ordered cache videos", result.List)
	}
	if result.List[1].PlayTime != "01:01" {
		t.Fatalf("playTime = %s, want 01:01", result.List[1].PlayTime)
	}
}

func TestLoadDynamicFeedFallsBackAndWritesRedisCacheOnMiss(t *testing.T) {
	repo := &fakeDynamicRepository{
		feed: []domain.WebVideoItem{
			{VideoID: "Video00003", UserID: "2000000001", LastUpdateTime: "2026-06-25 10:03:00", Duration: 65},
		},
	}
	feedCache := &fakeDynamicFeedCache{page: cache.DynamicFeedCachePage{Hit: false}}
	service := NewDynamicService(repo, feedCache)

	result, err := service.LoadFeed(context.Background(), LoadDynamicFeedInput{
		UserID:   "1000000001",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("load feed returned error: %v", err)
	}
	if repo.listFeedCalls != 1 {
		t.Fatalf("database feed calls = %d, want 1", repo.listFeedCalls)
	}
	if len(result.List) != 1 {
		t.Fatalf("feed len = %d, want 1", len(result.List))
	}
	if feedCache.addedUserID != "1000000001" || len(feedCache.addedItems) != 1 || feedCache.addedItems[0].VideoID != "Video00003" {
		t.Fatalf("cache added = user %s items %#v, want Video00003", feedCache.addedUserID, feedCache.addedItems)
	}
}

func TestLoadDynamicFeedIgnoresRedisWriteFailure(t *testing.T) {
	repo := &fakeDynamicRepository{
		feed: []domain.WebVideoItem{
			{VideoID: "Video00003", UserID: "2000000001", LastUpdateTime: "2026-06-25 10:03:00", Duration: 65},
		},
	}
	feedCache := &fakeDynamicFeedCache{
		page:   cache.DynamicFeedCachePage{Hit: false},
		addErr: errors.New("redis write failed"),
	}
	service := NewDynamicService(repo, feedCache)

	result, err := service.LoadFeed(context.Background(), LoadDynamicFeedInput{
		UserID:   "1000000001",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("load feed returned error: %v", err)
	}
	if len(result.List) != 1 || result.List[0].VideoID != "Video00003" {
		t.Fatalf("feed list = %#v, want database result", result.List)
	}
}

func TestLoadDynamicFeedRemovesDirtyRedisVideoID(t *testing.T) {
	repo := &fakeDynamicRepository{
		detailByID: map[string]domain.WebVideoItem{
			"Video00002": {VideoID: "Video00002", UserID: "2000000001", LastUpdateTime: "2026-06-25 10:02:00", Duration: 61},
		},
	}
	feedCache := &fakeDynamicFeedCache{
		page: cache.DynamicFeedCachePage{
			Hit: true,
			Items: []cache.DynamicFeedCacheItem{
				{VideoID: "Deleted001", Score: 1782352980000},
				{VideoID: "Video00002", Score: 1782352920000},
			},
		},
	}
	service := NewDynamicService(repo, feedCache)

	result, err := service.LoadFeed(context.Background(), LoadDynamicFeedInput{
		UserID:   "1000000001",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("load feed returned error: %v", err)
	}
	if len(result.List) != 1 || result.List[0].VideoID != "Video00002" {
		t.Fatalf("feed list = %#v, want Video00002", result.List)
	}
	if len(feedCache.removedVideoIDs) != 1 || feedCache.removedVideoIDs[0] != "Deleted001" {
		t.Fatalf("removed video ids = %#v, want Deleted001", feedCache.removedVideoIDs)
	}
}

type fakeDynamicRepository struct {
	currentUserInfo *domain.DynamicCurrentUserInfo
	followUsers     []domain.DynamicFollowUserItem
	feed            []domain.WebVideoItem
	detailByID      map[string]domain.WebVideoItem
	lastQuery       repository.DynamicFeedQuery
	listFeedCalls   int
	upsertedUserID  string
	upsertedFeed    []domain.WebVideoItem
}

func (r *fakeDynamicRepository) FindCurrentUserInfo(ctx context.Context, userID string) (*domain.DynamicCurrentUserInfo, error) {
	return r.currentUserInfo, nil
}

func (r *fakeDynamicRepository) ListFollowUsers(ctx context.Context, userID string) ([]domain.DynamicFollowUserItem, error) {
	return r.followUsers, nil
}

func (r *fakeDynamicRepository) ListFeedByCursor(ctx context.Context, query repository.DynamicFeedQuery) ([]domain.WebVideoItem, error) {
	r.lastQuery = query
	r.listFeedCalls++
	return r.feed, nil
}

func (r *fakeDynamicRepository) ListFeedVideoDetailsByIDs(ctx context.Context, videoIDs []string) ([]domain.WebVideoItem, error) {
	list := make([]domain.WebVideoItem, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		if item, ok := r.detailByID[videoID]; ok {
			list = append(list, item)
		}
	}
	return list, nil
}

func (r *fakeDynamicRepository) UpsertFeedItems(ctx context.Context, userID string, list []domain.WebVideoItem) error {
	r.upsertedUserID = userID
	r.upsertedFeed = append([]domain.WebVideoItem(nil), list...)
	return nil
}

type fakeDynamicFeedCache struct {
	page            cache.DynamicFeedCachePage
	addErr          error
	addedUserID     string
	addedItems      []cache.DynamicFeedCacheItem
	removedVideoIDs []string
}

func (c *fakeDynamicFeedCache) ListFeedItems(ctx context.Context, userID string, authorUserID string, cursor cache.DynamicFeedCacheCursor, limit int) (cache.DynamicFeedCachePage, error) {
	return c.page, nil
}

func (c *fakeDynamicFeedCache) AddFeedItems(ctx context.Context, userID string, items []cache.DynamicFeedCacheItem) error {
	c.addedUserID = userID
	c.addedItems = append([]cache.DynamicFeedCacheItem(nil), items...)
	return c.addErr
}

func (c *fakeDynamicFeedCache) RemoveFeedVideoIDs(ctx context.Context, userID string, authorUserID string, videoIDs []string) error {
	c.removedVideoIDs = append([]string(nil), videoIDs...)
	return nil
}
