package web

import (
	"context"
	"reflect"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
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

type fakeDynamicRepository struct {
	followUsers []domain.DynamicFollowUserItem
	feed        []domain.WebVideoItem
	lastQuery   repository.DynamicFeedQuery
}

func (r *fakeDynamicRepository) ListFollowUsers(ctx context.Context, userID string) ([]domain.DynamicFollowUserItem, error) {
	return r.followUsers, nil
}

func (r *fakeDynamicRepository) ListFeedByCursor(ctx context.Context, query repository.DynamicFeedQuery) ([]domain.WebVideoItem, error) {
	r.lastQuery = query
	return r.feed, nil
}
