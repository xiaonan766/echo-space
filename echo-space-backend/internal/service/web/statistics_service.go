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
	ListByUserTypeAndDateRange(ctx context.Context, userID string, dataType int, startDate string, endDate string) ([]domain.StatisticsInfo, error)
	ListCommentDailyCountByUserAndDateRange(ctx context.Context, userID string, dataType int, startDate string, endDate string) ([]domain.StatisticsInfo, error)
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

func (s *StatisticsService) GetWeekStatisticsInfo(ctx context.Context, userID string, dataType int) ([]domain.StatisticsInfo, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, &BusinessError{Info: "请先登录"}
	}
	if !validStatisticsType(dataType) {
		return nil, &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("statistics service is not ready")
	}

	dates := s.previousSevenDates()
	var statisticsList []domain.StatisticsInfo
	var err error
	if dataType == statisticsTypeComment {
		statisticsList, err = s.repository.ListCommentDailyCountByUserAndDateRange(ctx, userID, dataType, dates[0], dates[len(dates)-1])
	} else {
		statisticsList, err = s.repository.ListByUserTypeAndDateRange(ctx, userID, dataType, dates[0], dates[len(dates)-1])
	}
	if err != nil {
		return nil, err
	}
	return buildWeekStatisticsList(userID, dataType, dates, statisticsList), nil
}

func (s *StatisticsService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *StatisticsService) previousSevenDates() []string {
	current := s.currentTime()
	dates := make([]string, 0, 7)
	for offset := 7; offset >= 1; offset-- {
		dates = append(dates, current.AddDate(0, 0, -offset).Format("2006-01-02"))
	}
	return dates
}

func validStatisticsType(dataType int) bool {
	switch dataType {
	case statisticsTypePlay,
		statisticsTypeFans,
		statisticsTypeLike,
		statisticsTypeCollect,
		statisticsTypeCoin,
		statisticsTypeComment,
		statisticsTypeDanmu:
		return true
	default:
		return false
	}
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

func buildWeekStatisticsList(userID string, dataType int, dates []string, list []domain.StatisticsInfo) []domain.StatisticsInfo {
	dataMap := make(map[string]domain.StatisticsInfo, len(list))
	for _, item := range list {
		dataMap[item.StatisticsDate] = item
	}

	result := make([]domain.StatisticsInfo, 0, len(dates))
	for _, date := range dates {
		item, ok := dataMap[date]
		if !ok {
			item = domain.StatisticsInfo{
				StatisticsDate:  date,
				UserID:          userID,
				DateType:        dataType,
				StatisticsCount: 0,
			}
		}
		result = append(result, item)
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
