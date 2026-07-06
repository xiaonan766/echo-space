package web

import (
	"context"
	"errors"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type fakeUcenterContentVideoRepository struct {
	userID string
	list   []domain.UcenterAllVideoItem
	err    error
}

func (r *fakeUcenterContentVideoRepository) ListUserAllVideo(ctx context.Context, userID string) ([]domain.UcenterAllVideoItem, error) {
	r.userID = userID
	return r.list, r.err
}

type fakeUcenterContentInteractRepository struct {
	commentQuery     repository.UcenterInteractListQuery
	commentList      []domain.UcenterCommentItem
	commentErr       error
	commentCalled    int
	danmuQuery       repository.UcenterInteractListQuery
	danmuList        []domain.UcenterDanmuItem
	danmuErr         error
	danmuCalled      int
	collectionQuery  repository.UhomeCollectionListQuery
	collectionList   []domain.UserCollectionItem
	collectionTotal  int64
	collectionErr    error
	collectionCalled int
}

func (r *fakeUcenterContentInteractRepository) ListUcenterCommentByCursor(ctx context.Context, query repository.UcenterInteractListQuery) ([]domain.UcenterCommentItem, error) {
	r.commentCalled++
	r.commentQuery = query
	return r.commentList, r.commentErr
}

func (r *fakeUcenterContentInteractRepository) ListUcenterDanmuByCursor(ctx context.Context, query repository.UcenterInteractListQuery) ([]domain.UcenterDanmuItem, error) {
	r.danmuCalled++
	r.danmuQuery = query
	return r.danmuList, r.danmuErr
}

func (r *fakeUcenterContentInteractRepository) ListUserCollectionByPage(ctx context.Context, query repository.UhomeCollectionListQuery) ([]domain.UserCollectionItem, int64, error) {
	r.collectionCalled++
	r.collectionQuery = query
	return r.collectionList, r.collectionTotal, r.collectionErr
}

func TestUcenterContentServiceLoadAllVideo(t *testing.T) {
	videoRepository := &fakeUcenterContentVideoRepository{
		list: []domain.UcenterAllVideoItem{{VideoID: "Abc123Def4", VideoName: "测试视频"}},
	}
	service := NewUcenterContentService(videoRepository, nil)

	result, err := service.LoadAllVideo(context.Background(), " 1000000001 ")
	if err != nil {
		t.Fatalf("LoadAllVideo error = %v", err)
	}
	if videoRepository.userID != "1000000001" {
		t.Fatalf("repository userID = %s, want %s", videoRepository.userID, "1000000001")
	}
	if len(result) != 1 || result[0].VideoName != "测试视频" {
		t.Fatalf("result = %#v, want one video", result)
	}
}

func TestUcenterContentServiceLoadAllVideoRejectsEmptyUserID(t *testing.T) {
	service := NewUcenterContentService(&fakeUcenterContentVideoRepository{}, nil)

	_, err := service.LoadAllVideo(context.Background(), " ")
	if err == nil {
		t.Fatal("LoadAllVideo error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
}

func TestUcenterContentServiceLoadCommentFirstPage(t *testing.T) {
	interactRepository := &fakeUcenterContentInteractRepository{
		commentList: []domain.UcenterCommentItem{
			{CommentID: 30, VideoID: "Abc123Def4"},
			{CommentID: 29, VideoID: "Abc123Def4"},
			{CommentID: 28, VideoID: "Abc123Def4"},
			{CommentID: 27, VideoID: "Abc123Def4"},
		},
	}
	service := NewUcenterContentService(nil, interactRepository)

	result, err := service.LoadComment(context.Background(), UcenterInteractListInput{
		UserID:   " 1000000001 ",
		VideoID:  " Abc123Def4 ",
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("LoadComment error = %v", err)
	}
	query := interactRepository.commentQuery
	if query.UserID != "1000000001" || query.VideoID != "Abc123Def4" {
		t.Fatalf("query = %#v, want trimmed user and video", query)
	}
	if query.CursorID != 0 || query.Direction != ucenterCursorDirectionNext || query.Limit != 4 {
		t.Fatalf("query cursor = %#v, want first page limit 4", query)
	}
	assertCommentIDs(t, result.List, []int{30, 29, 28})
	if !result.HasNext || result.HasPrev || result.NextCursor == "" || result.PrevCursor != "" {
		t.Fatalf("result cursor flags = %#v, want next only", result)
	}
	assertCursor(t, result.NextCursor, ucenterCursorKindComment, ucenterCursorDirectionNext, 28, "Abc123Def4")
}

func TestUcenterContentServiceLoadCommentNextPage(t *testing.T) {
	cursor, err := encodeUcenterCursor(ucenterCursorKindComment, ucenterCursorDirectionNext, 28, "Abc123Def4")
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	interactRepository := &fakeUcenterContentInteractRepository{
		commentList: []domain.UcenterCommentItem{
			{CommentID: 27, VideoID: "Abc123Def4"},
			{CommentID: 26, VideoID: "Abc123Def4"},
			{CommentID: 25, VideoID: "Abc123Def4"},
			{CommentID: 24, VideoID: "Abc123Def4"},
		},
	}
	service := NewUcenterContentService(nil, interactRepository)

	result, err := service.LoadComment(context.Background(), UcenterInteractListInput{
		UserID:   "1000000001",
		VideoID:  "Abc123Def4",
		Cursor:   cursor,
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("LoadComment error = %v", err)
	}
	query := interactRepository.commentQuery
	if query.CursorID != 28 || query.Direction != ucenterCursorDirectionNext || query.Limit != 4 {
		t.Fatalf("query cursor = %#v, want next after 28", query)
	}
	assertCommentIDs(t, result.List, []int{27, 26, 25})
	if !result.HasNext || !result.HasPrev || result.NextCursor == "" || result.PrevCursor == "" {
		t.Fatalf("result cursor flags = %#v, want both directions", result)
	}
	assertCursor(t, result.NextCursor, ucenterCursorKindComment, ucenterCursorDirectionNext, 25, "Abc123Def4")
	assertCursor(t, result.PrevCursor, ucenterCursorKindComment, ucenterCursorDirectionPrev, 27, "Abc123Def4")
}

func TestUcenterContentServiceLoadCommentPrevPage(t *testing.T) {
	cursor, err := encodeUcenterCursor(ucenterCursorKindComment, ucenterCursorDirectionPrev, 27, "Abc123Def4")
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	interactRepository := &fakeUcenterContentInteractRepository{
		commentList: []domain.UcenterCommentItem{
			{CommentID: 28, VideoID: "Abc123Def4"},
			{CommentID: 29, VideoID: "Abc123Def4"},
			{CommentID: 30, VideoID: "Abc123Def4"},
			{CommentID: 31, VideoID: "Abc123Def4"},
		},
	}
	service := NewUcenterContentService(nil, interactRepository)

	result, err := service.LoadComment(context.Background(), UcenterInteractListInput{
		UserID:   "1000000001",
		VideoID:  "Abc123Def4",
		Cursor:   cursor,
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("LoadComment error = %v", err)
	}
	query := interactRepository.commentQuery
	if query.CursorID != 27 || query.Direction != ucenterCursorDirectionPrev || query.Limit != 4 {
		t.Fatalf("query cursor = %#v, want prev before 27", query)
	}
	assertCommentIDs(t, result.List, []int{30, 29, 28})
	if !result.HasNext || !result.HasPrev || result.NextCursor == "" || result.PrevCursor == "" {
		t.Fatalf("result cursor flags = %#v, want both directions", result)
	}
	assertCursor(t, result.NextCursor, ucenterCursorKindComment, ucenterCursorDirectionNext, 28, "Abc123Def4")
	assertCursor(t, result.PrevCursor, ucenterCursorKindComment, ucenterCursorDirectionPrev, 30, "Abc123Def4")
}

func TestUcenterContentServiceLoadDanmuFirstNextAndPrevPage(t *testing.T) {
	tests := []struct {
		name          string
		cursor        string
		rawList       []domain.UcenterDanmuItem
		wantDirection string
		wantCursorID  int
		wantIDs       []int
		wantHasNext   bool
		wantHasPrev   bool
	}{
		{
			name: "first",
			rawList: []domain.UcenterDanmuItem{
				{DanmuID: 20, VideoID: "Abc123Def4"},
				{DanmuID: 19, VideoID: "Abc123Def4"},
				{DanmuID: 18, VideoID: "Abc123Def4"},
			},
			wantDirection: ucenterCursorDirectionNext,
			wantIDs:       []int{20, 19},
			wantHasNext:   true,
			wantHasPrev:   false,
		},
		{
			name:          "next",
			cursor:        mustEncodeUcenterCursor(t, ucenterCursorKindDanmu, ucenterCursorDirectionNext, 18, "Abc123Def4"),
			rawList:       []domain.UcenterDanmuItem{{DanmuID: 17}, {DanmuID: 16}, {DanmuID: 15}},
			wantDirection: ucenterCursorDirectionNext,
			wantCursorID:  18,
			wantIDs:       []int{17, 16},
			wantHasNext:   true,
			wantHasPrev:   true,
		},
		{
			name:          "prev",
			cursor:        mustEncodeUcenterCursor(t, ucenterCursorKindDanmu, ucenterCursorDirectionPrev, 17, "Abc123Def4"),
			rawList:       []domain.UcenterDanmuItem{{DanmuID: 18}, {DanmuID: 19}, {DanmuID: 20}},
			wantDirection: ucenterCursorDirectionPrev,
			wantCursorID:  17,
			wantIDs:       []int{19, 18},
			wantHasNext:   true,
			wantHasPrev:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interactRepository := &fakeUcenterContentInteractRepository{danmuList: tt.rawList}
			service := NewUcenterContentService(nil, interactRepository)

			result, err := service.LoadDanmu(context.Background(), UcenterInteractListInput{
				UserID:   "1000000001",
				VideoID:  "Abc123Def4",
				Cursor:   tt.cursor,
				PageSize: 2,
			})
			if err != nil {
				t.Fatalf("LoadDanmu error = %v", err)
			}
			query := interactRepository.danmuQuery
			if query.CursorID != tt.wantCursorID || query.Direction != tt.wantDirection || query.Limit != 3 {
				t.Fatalf("query cursor = %#v, want id=%d direction=%s limit=3", query, tt.wantCursorID, tt.wantDirection)
			}
			assertDanmuIDs(t, result.List, tt.wantIDs)
			if result.HasNext != tt.wantHasNext || result.HasPrev != tt.wantHasPrev {
				t.Fatalf("result cursor flags = %#v", result)
			}
		})
	}
}

func TestUcenterContentServiceRejectsInvalidCursor(t *testing.T) {
	interactRepository := &fakeUcenterContentInteractRepository{}
	service := NewUcenterContentService(nil, interactRepository)

	_, err := service.LoadComment(context.Background(), UcenterInteractListInput{
		UserID: "1000000001",
		Cursor: "bad-cursor",
	})
	if err == nil {
		t.Fatal("LoadComment error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
	if interactRepository.commentCalled != 0 {
		t.Fatalf("repository called = %d, want 0", interactRepository.commentCalled)
	}
}

func TestUcenterContentServiceRejectsCursorKindMismatch(t *testing.T) {
	cursor := mustEncodeUcenterCursor(t, ucenterCursorKindDanmu, ucenterCursorDirectionNext, 10, "")
	service := NewUcenterContentService(nil, &fakeUcenterContentInteractRepository{})

	_, err := service.LoadComment(context.Background(), UcenterInteractListInput{
		UserID: "1000000001",
		Cursor: cursor,
	})
	if err == nil {
		t.Fatal("LoadComment error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
}

func TestUcenterContentServiceRejectsCursorVideoMismatch(t *testing.T) {
	cursor := mustEncodeUcenterCursor(t, ucenterCursorKindComment, ucenterCursorDirectionNext, 10, "Abc123Def4")
	service := NewUcenterContentService(nil, &fakeUcenterContentInteractRepository{})

	_, err := service.LoadComment(context.Background(), UcenterInteractListInput{
		UserID:  "1000000001",
		VideoID: "Def123Ghi5",
		Cursor:  cursor,
	})
	if err == nil {
		t.Fatal("LoadComment error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
}

func TestUcenterContentServicePageSizeDefaultAndMax(t *testing.T) {
	interactRepository := &fakeUcenterContentInteractRepository{}
	service := NewUcenterContentService(nil, interactRepository)

	defaultResult, err := service.LoadDanmu(context.Background(), UcenterInteractListInput{UserID: "1000000001"})
	if err != nil {
		t.Fatalf("LoadDanmu default page size error = %v", err)
	}
	if defaultResult.PageSize != defaultUcenterContentPageSize || interactRepository.danmuQuery.Limit != defaultUcenterContentPageSize+1 {
		t.Fatalf("default page size result=%d limit=%d", defaultResult.PageSize, interactRepository.danmuQuery.Limit)
	}

	maxResult, err := service.LoadDanmu(context.Background(), UcenterInteractListInput{
		UserID:   "1000000001",
		PageSize: 500,
	})
	if err != nil {
		t.Fatalf("LoadDanmu max page size error = %v", err)
	}
	if maxResult.PageSize != maxUcenterContentPageSize || interactRepository.danmuQuery.Limit != maxUcenterContentPageSize+1 {
		t.Fatalf("max page size result=%d limit=%d", maxResult.PageSize, interactRepository.danmuQuery.Limit)
	}
}

func TestUcenterContentServiceLoadDanmuRejectsInvalidVideoID(t *testing.T) {
	service := NewUcenterContentService(nil, &fakeUcenterContentInteractRepository{})

	_, err := service.LoadDanmu(context.Background(), UcenterInteractListInput{
		UserID:  "1000000001",
		VideoID: "bad",
	})
	if err == nil {
		t.Fatal("LoadDanmu error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
}

func TestUcenterContentServiceLoadDanmuReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("db error")
	service := NewUcenterContentService(nil, &fakeUcenterContentInteractRepository{danmuErr: expectedErr})

	_, err := service.LoadDanmu(context.Background(), UcenterInteractListInput{UserID: "1000000001"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestUcenterContentServiceLoadUserCollection(t *testing.T) {
	interactRepository := &fakeUcenterContentInteractRepository{
		collectionList: []domain.UserCollectionItem{
			{ActionID: 20, VideoID: "Abc123Def4", VideoName: "测试收藏"},
		},
		collectionTotal: 21,
	}
	service := NewUcenterContentService(nil, interactRepository)

	result, err := service.LoadUserCollection(context.Background(), UhomeCollectionListInput{
		UserID:   " 1000000001 ",
		PageNo:   2,
		PageSize: 8,
	})
	if err != nil {
		t.Fatalf("LoadUserCollection error = %v", err)
	}
	query := interactRepository.collectionQuery
	if query.UserID != "1000000001" || query.PageNo != 2 || query.PageSize != 8 {
		t.Fatalf("query = %#v, want trimmed user and page", query)
	}
	if result.TotalCount != 21 || result.PageNo != 2 || result.PageSize != 8 || result.PageTotal != 3 {
		t.Fatalf("pagination result = %#v, want total 21 page 2 size 8 totalPage 3", result)
	}
	if len(result.List) != 1 || result.List[0].VideoName != "测试收藏" {
		t.Fatalf("result list = %#v, want one collection item", result.List)
	}
}

func TestUcenterContentServiceLoadUserCollectionRejectsInvalidUserID(t *testing.T) {
	interactRepository := &fakeUcenterContentInteractRepository{}
	service := NewUcenterContentService(nil, interactRepository)

	_, err := service.LoadUserCollection(context.Background(), UhomeCollectionListInput{UserID: "bad"})
	if err == nil {
		t.Fatal("LoadUserCollection error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
	if interactRepository.collectionCalled != 0 {
		t.Fatalf("repository called = %d, want 0", interactRepository.collectionCalled)
	}
}

func TestUcenterContentServiceLoadUserCollectionDefaultsPage(t *testing.T) {
	interactRepository := &fakeUcenterContentInteractRepository{}
	service := NewUcenterContentService(nil, interactRepository)

	result, err := service.LoadUserCollection(context.Background(), UhomeCollectionListInput{
		UserID:   "1000000001",
		PageNo:   -1,
		PageSize: 500,
	})
	if err != nil {
		t.Fatalf("LoadUserCollection error = %v", err)
	}
	query := interactRepository.collectionQuery
	if query.PageNo != defaultUhomeCollectionPageNo || query.PageSize != maxUhomeCollectionPageSize {
		t.Fatalf("query = %#v, want default pageNo and max pageSize", query)
	}
	if result.PageNo != defaultUhomeCollectionPageNo || result.PageSize != maxUhomeCollectionPageSize {
		t.Fatalf("result = %#v, want normalized pagination", result)
	}
}

func assertCommentIDs(t *testing.T, list []domain.UcenterCommentItem, want []int) {
	t.Helper()
	if len(list) != len(want) {
		t.Fatalf("comment list length = %d, want %d: %#v", len(list), len(want), list)
	}
	for i, item := range list {
		if item.CommentID != want[i] {
			t.Fatalf("comment list[%d].CommentID = %d, want %d", i, item.CommentID, want[i])
		}
	}
}

func assertDanmuIDs(t *testing.T, list []domain.UcenterDanmuItem, want []int) {
	t.Helper()
	if len(list) != len(want) {
		t.Fatalf("danmu list length = %d, want %d: %#v", len(list), len(want), list)
	}
	for i, item := range list {
		if item.DanmuID != want[i] {
			t.Fatalf("danmu list[%d].DanmuID = %d, want %d", i, item.DanmuID, want[i])
		}
	}
}

func assertCursor(t *testing.T, cursor string, kind string, direction string, anchorID int, videoID string) {
	t.Helper()
	payload, hasCursor, err := decodeUcenterCursor(cursor, kind, videoID)
	if err != nil {
		t.Fatalf("decode cursor error = %v", err)
	}
	if !hasCursor {
		t.Fatal("hasCursor = false, want true")
	}
	if payload.Kind != kind || payload.Direction != direction || payload.AnchorID != anchorID || payload.VideoID != videoID {
		t.Fatalf("cursor payload = %#v, want kind=%s direction=%s anchor=%d video=%s", payload, kind, direction, anchorID, videoID)
	}
}

func mustEncodeUcenterCursor(t *testing.T, kind string, direction string, anchorID int, videoID string) string {
	t.Helper()
	cursor, err := encodeUcenterCursor(kind, direction, anchorID, videoID)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	return cursor
}
