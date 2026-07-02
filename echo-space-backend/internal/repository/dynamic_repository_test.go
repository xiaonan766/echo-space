package repository

import (
	"reflect"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

func TestDynamicFeedIDUsesImagePrefix(t *testing.T) {
	if got := dynamicFeedID("1000000001", domain.DynamicEventTypeVideo, "Video00001"); got != "1000000001_Video00001" {
		t.Fatalf("video feed id = %s, want legacy video format", got)
	}
	if got := dynamicFeedID("1000000001", domain.DynamicEventTypeImage, "Image00001"); got != "1000000001_image_Image00001" {
		t.Fatalf("image feed id = %s, want image-prefixed format", got)
	}
}

func TestSplitDynamicFeedContentKeys(t *testing.T) {
	videoIDs, imageIDs := splitDynamicFeedContentKeys([]domain.DynamicFeedContentKey{
		{ContentType: domain.ContentTypeVideo, ContentID: "Video00001"},
		{ContentType: domain.ContentTypeImage, ContentID: "Image00001"},
		{ContentType: domain.ContentTypeImage, ContentID: "Image00001"},
	})
	if !reflect.DeepEqual(videoIDs, []string{"Video00001"}) {
		t.Fatalf("video ids = %#v, want one video id", videoIDs)
	}
	if !reflect.DeepEqual(imageIDs, []string{"Image00001"}) {
		t.Fatalf("image ids = %#v, want de-duplicated image id", imageIDs)
	}
}
