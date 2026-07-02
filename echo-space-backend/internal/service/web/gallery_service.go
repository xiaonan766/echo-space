package web

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultGalleryPageNo   = 1
	defaultGalleryPageSize = 15
	maxGalleryPageSize     = 50
)

type GalleryImageListInput struct {
	PageNo   int
	PageSize int
}

type galleryRepository interface {
	ListApprovedImagesByPage(ctx context.Context, query repository.GalleryImageListQuery) ([]domain.GalleryImageItem, int64, error)
	FindApprovedImageByID(ctx context.Context, imageID string) (*domain.GalleryImageInfo, error)
	ListApprovedImageFiles(ctx context.Context, imageID string) ([]domain.GalleryImageFile, error)
}

type GalleryService struct {
	galleryRepository galleryRepository
}

func NewGalleryService(galleryRepository galleryRepository) *GalleryService {
	return &GalleryService{galleryRepository: galleryRepository}
}

func (s *GalleryService) LoadImageList(ctx context.Context, input GalleryImageListInput) (domain.PaginationResult[domain.GalleryImageItem], error) {
	input = normalizeGalleryImageListInput(input)
	if s == nil || s.galleryRepository == nil {
		return domain.PaginationResult[domain.GalleryImageItem]{}, errors.New("gallery service is not ready")
	}

	list, totalCount, err := s.galleryRepository.ListApprovedImagesByPage(ctx, repository.GalleryImageListQuery{
		PageNo:   input.PageNo,
		PageSize: input.PageSize,
	})
	if err != nil {
		return domain.PaginationResult[domain.GalleryImageItem]{}, err
	}
	return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
}

func (s *GalleryService) GetImageInfo(ctx context.Context, imageID string) (domain.GalleryImageDetail, error) {
	imageID = strings.TrimSpace(imageID)
	if len(imageID) != videoIDLength || !isValidPublicVideoID(imageID) {
		return domain.GalleryImageDetail{}, &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.galleryRepository == nil {
		return domain.GalleryImageDetail{}, errors.New("gallery service is not ready")
	}

	imageInfo, err := s.galleryRepository.FindApprovedImageByID(ctx, imageID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.GalleryImageDetail{}, &BusinessError{Info: "图片不存在"}
	}
	if err != nil {
		return domain.GalleryImageDetail{}, err
	}
	imageList, err := s.galleryRepository.ListApprovedImageFiles(ctx, imageID)
	if err != nil {
		return domain.GalleryImageDetail{}, err
	}
	return domain.GalleryImageDetail{
		ImageInfo: *imageInfo,
		ImageList: imageList,
	}, nil
}

func normalizeGalleryImageListInput(input GalleryImageListInput) GalleryImageListInput {
	if input.PageNo <= 0 {
		input.PageNo = defaultGalleryPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultGalleryPageSize
	}
	if input.PageSize > maxGalleryPageSize {
		input.PageSize = maxGalleryPageSize
	}
	return input
}
