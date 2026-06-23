package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultVideoPageNo   = 1
	defaultVideoPageSize = 15
	maxVideoPageSize     = 50
	videoNoRecommend     = 0
	videoRecommend       = 1
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

	fillWebVideoPlayTime(list)
	return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
}

func (s *VideoService) LoadRecommendVideo(ctx context.Context) ([]domain.WebVideoItem, error) {
	list, err := s.videoRepository.ListWebVideo(ctx, repository.WebVideoListQuery{
		RecommendType: videoRecommend,
	})
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.WebVideoItem{}
	}
	fillWebVideoPlayTime(list)
	return list, nil
}

func (s *VideoService) LoadVideoPList(ctx context.Context, videoID string) ([]domain.VideoInfoFile, error) {
	videoID = strings.TrimSpace(videoID)
	if len(videoID) != 10 || !isValidPublicVideoID(videoID) {
		return nil, &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	files, err := s.videoRepository.ListVideoFiles(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = []domain.VideoInfoFile{}
	}
	return files, nil
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

func fillWebVideoPlayTime(list []domain.WebVideoItem) {
	for index := range list {
		list[index].PlayTime = formatVideoDuration(list[index].Duration)
	}
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

func isValidPublicVideoID(videoID string) bool {
	for _, char := range videoID {
		if char >= '0' && char <= '9' {
			continue
		}
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		return false
	}
	return true
}
