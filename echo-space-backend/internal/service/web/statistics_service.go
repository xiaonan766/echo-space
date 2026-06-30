package web

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const (
	statisticsTypePlay    = 0
	statisticsTypeFans    = 1
	statisticsTypeLike    = 2
	statisticsTypeCollect = 3
	statisticsTypeCoin    = 4
	statisticsTypeComment = 5
	statisticsTypeDanmu   = 6
)

type StatisticsDataRepository interface {
	ListByUserAndDate(ctx context.Context, userID string, statisticsDate string) ([]domain.StatisticsInfo, error)
	GetTotalStatisticsCountInfo(ctx context.Context, userID string) (map[string]int, error)
}

type StatisticsService struct {
	repository StatisticsDataRepository
	now        func() time.Time
}

func NewStatisticsService(repository StatisticsDataRepository) *StatisticsService {
	return &StatisticsService{
		repository: repository,
		now:        time.Now,
	}
}

func (s *StatisticsService) GetActualTimeStatisticsInfo(ctx context.Context, userID string) (*domain.ActualTimeStatisticsInfo, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, &BusinessError{Info: "请先登录"}
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("statistics service is not ready")
	}

	beforeDay := s.currentTime().AddDate(0, 0, -1).Format("2006-01-02")
	statisticsList, err := s.repository.ListByUserAndDate(ctx, userID, beforeDay)
	if err != nil {
		return nil, err
	}
	totalCountInfo, err := s.repository.GetTotalStatisticsCountInfo(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &domain.ActualTimeStatisticsInfo{
		PreDayData:     buildPreDayStatisticsMap(statisticsList),
		TotalCountInfo: normalizeTotalStatisticsMap(totalCountInfo),
	}, nil
}

func (s *StatisticsService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func buildPreDayStatisticsMap(list []domain.StatisticsInfo) map[int]int {
	result := map[int]int{
		statisticsTypePlay:    0,
		statisticsTypeFans:    0,
		statisticsTypeLike:    0,
		statisticsTypeCollect: 0,
		statisticsTypeCoin:    0,
		statisticsTypeComment: 0,
		statisticsTypeDanmu:   0,
	}
	for _, item := range list {
		if _, ok := result[item.DateType]; ok {
			result[item.DateType] = item.StatisticsCount
		}
	}
	return result
}

func normalizeTotalStatisticsMap(input map[string]int) map[string]int {
	result := map[string]int{
		"userCount":    0,
		"playCount":    0,
		"commentCount": 0,
		"danmuCount":   0,
		"likeCount":    0,
		"collectCount": 0,
		"coinCount":    0,
	}
	for key, value := range input {
		if _, ok := result[key]; ok {
			result[key] = value
		}
	}
	return result
}
