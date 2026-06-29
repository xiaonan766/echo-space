package web

import (
	"context"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	searchinfra "github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/search"
)

func TestSearchVideoEmptyKeywordReturnsEmptyPagination(t *testing.T) {
	service := NewVideoService(nil)

	result, err := service.SearchVideo(context.Background(), VideoSearchInput{
		Keyword:  "   ",
		PageNo:   0,
		PageSize: 0,
	})
	if err != nil {
		t.Fatalf("SearchVideo returned error: %v", err)
	}
	if result.TotalCount != 0 || len(result.List) != 0 {
		t.Fatalf("result = %#v, want empty pagination", result)
	}
	if result.PageNo != defaultVideoPageNo {
		t.Fatalf("pageNo = %d, want %d", result.PageNo, defaultVideoPageNo)
	}
	if result.PageSize != defaultSearchPageSize {
		t.Fatalf("pageSize = %d, want %d", result.PageSize, defaultSearchPageSize)
	}
}

func TestReorderSearchVideoListEscapesPlainTitle(t *testing.T) {
	list := []domain.WebVideoItem{
		{VideoID: "video1", VideoName: "<script>alert(1)</script>"},
	}
	hits := []searchinfra.VideoSearchHit{
		{VideoID: "video1"},
	}

	result := reorderSearchVideoList(list, hits)
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	want := "&lt;script&gt;alert(1)&lt;/script&gt;"
	if result[0].VideoName != want {
		t.Fatalf("videoName = %q, want %q", result[0].VideoName, want)
	}
}

func TestReorderSearchVideoListKeepsHighlightedTitle(t *testing.T) {
	list := []domain.WebVideoItem{
		{VideoID: "video1", VideoName: "Vue demo"},
	}
	hits := []searchinfra.VideoSearchHit{
		{VideoID: "video1", HighlightName: "<span class='highlight'>Vue</span> demo"},
	}

	result := reorderSearchVideoList(list, hits)
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	want := "<span class='highlight'>Vue</span> demo"
	if result[0].VideoName != want {
		t.Fatalf("videoName = %q, want %q", result[0].VideoName, want)
	}
}
