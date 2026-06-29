package web

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	searchinfra "github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/search"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultVideoPageNo     = 1
	defaultVideoPageSize   = 15
	defaultSearchPageSize  = 30
	maxVideoPageSize       = 50
	maxSearchKeywordLength = 100
	videoNoRecommend       = 0
	videoRecommend         = 1
)

type VideoService struct {
	videoRepository *repository.VideoRepository
	videoSearch     *searchinfra.VideoIndex
	keywordStore    *cache.SearchKeywordStore
}

type VideoListInput struct {
	PageNo      int
	PageSize    int
	PCategoryID int
	CategoryID  int
}

type VideoSearchInput struct {
	Keyword   string
	OrderType *int
	PageNo    int
	PageSize  int
}

type VideoServiceOption func(*VideoService)

func WithVideoSearch(videoSearch *searchinfra.VideoIndex) VideoServiceOption {
	return func(service *VideoService) {
		service.videoSearch = videoSearch
	}
}

func WithSearchKeywordStore(keywordStore *cache.SearchKeywordStore) VideoServiceOption {
	return func(service *VideoService) {
		service.keywordStore = keywordStore
	}
}

func NewVideoService(videoRepository *repository.VideoRepository, options ...VideoServiceOption) *VideoService {
	service := &VideoService{videoRepository: videoRepository}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
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

func (s *VideoService) SearchVideo(ctx context.Context, input VideoSearchInput) (domain.PaginationResult[domain.WebVideoItem], error) {
	input = normalizeVideoSearchInput(input)
	if input.Keyword == "" {
		return domain.NewPaginationResult([]domain.WebVideoItem{}, 0, input.PageNo, input.PageSize), nil
	}
	if utf8.RuneCountInString(input.Keyword) > maxSearchKeywordLength {
		return domain.PaginationResult[domain.WebVideoItem]{}, &BusinessError{Info: "\u641c\u7d22\u5173\u952e\u8bcd\u4e0d\u80fd\u8d85\u8fc7100\u4e2a\u5b57\u7b26"}
	}
	if s == nil || s.videoRepository == nil || s.videoSearch == nil {
		return domain.PaginationResult[domain.WebVideoItem]{}, errors.New("video search service is not ready")
	}

	if s.keywordStore != nil {
		if err := s.keywordStore.Add(ctx, input.Keyword); err != nil {
			log.Printf("record search keyword failed: %v", err)
		}
	}

	searchResult, err := s.videoSearch.Search(ctx, searchinfra.VideoSearchInput{
		Keyword:   input.Keyword,
		OrderType: input.OrderType,
		PageNo:    input.PageNo,
		PageSize:  input.PageSize,
		Highlight: true,
	})
	if err != nil {
		return domain.PaginationResult[domain.WebVideoItem]{}, err
	}

	videoIDs := collectSearchVideoIDs(searchResult.Hits)
	videoList, err := s.videoRepository.ListWebVideoByIDs(ctx, videoIDs)
	if err != nil {
		return domain.PaginationResult[domain.WebVideoItem]{}, err
	}
	videoList = reorderSearchVideoList(videoList, searchResult.Hits)
	fillWebVideoPlayTime(videoList)
	return domain.NewPaginationResult(videoList, searchResult.TotalCount, input.PageNo, input.PageSize), nil
}

func (s *VideoService) GetVideoInfo(ctx context.Context, videoID string, userID string) (domain.WebVideoDetail, error) {
	videoID = strings.TrimSpace(videoID)
	userID = strings.TrimSpace(userID)
	if len(videoID) != 10 || !isValidPublicVideoID(videoID) {
		return domain.WebVideoDetail{}, &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	videoInfo, err := s.videoRepository.FindWebVideoByID(ctx, videoID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.WebVideoDetail{}, &BusinessError{Info: "\u89c6\u9891\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return domain.WebVideoDetail{}, err
	}
	videoList := []domain.WebVideoItem{*videoInfo}
	fillWebVideoPlayTime(videoList)

	userActionList, err := s.videoRepository.ListUserVideoActions(ctx, videoID, userID)
	if err != nil {
		return domain.WebVideoDetail{}, err
	}

	return domain.WebVideoDetail{
		VideoInfo:      videoList[0],
		UserActionList: userActionList,
	}, nil
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

func normalizeVideoSearchInput(input VideoSearchInput) VideoSearchInput {
	input.Keyword = strings.TrimSpace(input.Keyword)
	if input.PageNo <= 0 {
		input.PageNo = defaultVideoPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultSearchPageSize
	}
	if input.PageSize > maxVideoPageSize {
		input.PageSize = maxVideoPageSize
	}
	if input.OrderType != nil {
		orderType := *input.OrderType
		if _, ok := searchinfra.SearchOrderField(orderType); ok {
			input.OrderType = &orderType
		} else {
			input.OrderType = nil
		}
	}
	return input
}

func collectSearchVideoIDs(hits []searchinfra.VideoSearchHit) []string {
	videoIDs := make([]string, 0, len(hits))
	seen := make(map[string]struct{}, len(hits))
	for _, hit := range hits {
		videoID := strings.TrimSpace(hit.VideoID)
		if videoID == "" {
			continue
		}
		if _, exists := seen[videoID]; exists {
			continue
		}
		seen[videoID] = struct{}{}
		videoIDs = append(videoIDs, videoID)
	}
	return videoIDs
}

func reorderSearchVideoList(videoList []domain.WebVideoItem, hits []searchinfra.VideoSearchHit) []domain.WebVideoItem {
	videoMap := make(map[string]domain.WebVideoItem, len(videoList))
	for _, item := range videoList {
		item.VideoName = html.EscapeString(item.VideoName)
		videoMap[item.VideoID] = item
	}

	result := make([]domain.WebVideoItem, 0, len(videoList))
	for _, hit := range hits {
		item, ok := videoMap[hit.VideoID]
		if !ok {
			continue
		}
		if hit.HighlightName != "" {
			item.VideoName = hit.HighlightName
		}
		result = append(result, item)
		delete(videoMap, hit.VideoID)
	}
	return result
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
