package web

import (
	"context"
	"fmt"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultVideoPageNo   = 1
	defaultVideoPageSize = 15
	maxVideoPageSize     = 50
	videoNoRecommend     = 0
)

type VideoService struct {
	videoRepository *repository.VideoRepository
}

type VideoListInput struct {
	PageNo      int
	PageSize    int
	PCategoryID int
	CategoryID  int
}

func NewVideoService(videoRepository *repository.VideoRepository) *VideoService {
	return &VideoService{videoRepository: videoRepository}
}

func (s *VideoService) LoadVideo(ctx context.Context, input VideoListInput) (domain.PaginationResult[domain.WebVideoItem], error) {
	input = normalizeVideoListInput(input)
	list, totalCount, err := s.videoRepository.ListWebVideoByPage(ctx, repository.WebVideoListQuery{
		PageNo:        input.PageNo,
		PageSize:      input.PageSize,
		PCategoryID:   input.PCategoryID,
		CategoryID:    input.CategoryID,
		RecommendType: videoNoRecommend,
	})
	if err != nil {
		return domain.PaginationResult[domain.WebVideoItem]{}, err
	}

	for index := range list {
		list[index].PlayTime = formatVideoDuration(list[index].Duration)
	}
	return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
}

func normalizeVideoListInput(input VideoListInput) VideoListInput {
	if input.PageNo <= 0 {
		input.PageNo = defaultVideoPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultVideoPageSize
	}
	if input.PageSize > maxVideoPageSize {
		input.PageSize = maxVideoPageSize
	}
	if input.PCategoryID < 0 {
		input.PCategoryID = 0
	}
	if input.CategoryID < 0 {
		input.CategoryID = 0
	}
	return input
}

func formatVideoDuration(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}

	hours := seconds / 3600
	minutes := seconds % 3600 / 60
	remainingSeconds := seconds % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, remainingSeconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, remainingSeconds)
}
