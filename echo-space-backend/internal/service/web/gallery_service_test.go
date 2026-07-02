package web

import (
	"context"
	"errors"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type fakeGalleryRepository struct {
	listQuery repository.GalleryImageListQuery
}

func (r *fakeGalleryRepository) ListApprovedImagesByPage(ctx context.Context, query repository.GalleryImageListQuery) ([]domain.GalleryImageItem, int64, error) {
	r.listQuery = query
	return []domain.GalleryImageItem{{ImageID: "Abc123Def4"}}, 1, nil
}

func (r *fakeGalleryRepository) FindApprovedImageByID(ctx context.Context, imageID string) (*domain.GalleryImageInfo, error) {
	if imageID == "NotFound01" {
		return nil, errors.New("unexpected")
	}
	return &domain.GalleryImageInfo{ImageID: imageID, ImageName: "测试图片"}, nil
}

func (r *fakeGalleryRepository) ListApprovedImageFiles(ctx context.Context, imageID string) ([]domain.GalleryImageFile, error) {
	return []domain.GalleryImageFile{{FileID: "File001", SourceName: "images/202607/image.png"}}, nil
}

func TestGalleryServiceLoadImageListNormalizesPagination(t *testing.T) {
	repo := &fakeGalleryRepository{}
	service := NewGalleryService(repo)

	result, err := service.LoadImageList(context.Background(), GalleryImageListInput{
		PageNo:   -1,
		PageSize: 500,
	})
	if err != nil {
		t.Fatalf("LoadImageList returned error: %v", err)
	}
	if repo.listQuery.PageNo != defaultGalleryPageNo {
		t.Fatalf("PageNo = %d, want %d", repo.listQuery.PageNo, defaultGalleryPageNo)
	}
	if repo.listQuery.PageSize != maxGalleryPageSize {
		t.Fatalf("PageSize = %d, want %d", repo.listQuery.PageSize, maxGalleryPageSize)
	}
	if result.PageSize != maxGalleryPageSize || result.TotalCount != 1 {
		t.Fatalf("unexpected pagination result: %+v", result)
	}
}

func TestGalleryServiceGetImageInfoValidatesImageID(t *testing.T) {
	service := NewGalleryService(&fakeGalleryRepository{})

	_, err := service.GetImageInfo(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if businessError, ok := IsBusinessError(err); !ok || businessError.Info != "参数错误" {
		t.Fatalf("err = %v, want 参数错误 business error", err)
	}
}

func TestGalleryServiceGetImageInfoReturnsFiles(t *testing.T) {
	service := NewGalleryService(&fakeGalleryRepository{})

	result, err := service.GetImageInfo(context.Background(), "Abc123Def4")
	if err != nil {
		t.Fatalf("GetImageInfo returned error: %v", err)
	}
	if result.ImageInfo.ImageID != "Abc123Def4" {
		t.Fatalf("ImageID = %q, want Abc123Def4", result.ImageInfo.ImageID)
	}
	if len(result.ImageList) != 1 || result.ImageList[0].SourceName == "" {
		t.Fatalf("unexpected image files: %+v", result.ImageList)
	}
}
