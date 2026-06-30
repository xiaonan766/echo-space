package web

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type fakeStatisticsRepository struct {
	list           []domain.StatisticsInfo
	total          map[string]int
	listErr        error
	totalErr       error
	userID         string
	statisticsDate string
}

func (r *fakeStatisticsRepository) ListByUserAndDate(ctx context.Context, userID string, statisticsDate string) ([]domain.StatisticsInfo, error) {
	r.userID = userID
	r.statisticsDate = statisticsDate
	return r.list, r.listErr
}

func (r *fakeStatisticsRepository) GetTotalStatisticsCountInfo(ctx context.Context, userID string) (map[string]int, error) {
	r.userID = userID
	return r.total, r.totalErr
}

func TestStatisticsServiceGetActualTimeStatisticsInfo(t *testing.T) {
	repository := &fakeStatisticsRepository{
		list: []domain.StatisticsInfo{
			{DateType: statisticsTypePlay, StatisticsCount: 12},
			{DateType: statisticsTypeFans, StatisticsCount: 3},
			{DateType: 99, StatisticsCount: 100},
		},
		total: map[string]int{
			"userCount": 5,
			"playCount": 20,
			"likeCount": 8,
		},
	}
	service := NewStatisticsService(repository)
	service.now = func() time.Time {
		return time.Date(2026, 6, 30, 12, 0, 0, 0, time.Local)
	}

	result, err := service.GetActualTimeStatisticsInfo(context.Background(), "1000000001")
	if err != nil {
		t.Fatalf("GetActualTimeStatisticsInfo error = %v", err)
	}
	if repository.userID != "1000000001" {
		t.Fatalf("repository userID = %s, want %s", repository.userID, "1000000001")
	}
	if repository.statisticsDate != "2026-06-29" {
		t.Fatalf("statisticsDate = %s, want %s", repository.statisticsDate, "2026-06-29")
	}
	if result.PreDayData[statisticsTypePlay] != 12 {
		t.Fatalf("preDay play = %d, want %d", result.PreDayData[statisticsTypePlay], 12)
	}
	if result.PreDayData[statisticsTypeFans] != 3 {
		t.Fatalf("preDay fans = %d, want %d", result.PreDayData[statisticsTypeFans], 3)
	}
	if _, ok := result.PreDayData[99]; ok {
		t.Fatalf("unexpected date type 99 in preDayData")
	}
	if result.TotalCountInfo["userCount"] != 5 {
		t.Fatalf("total userCount = %d, want %d", result.TotalCountInfo["userCount"], 5)
	}
	if result.TotalCountInfo["danmuCount"] != 0 {
		t.Fatalf("total danmuCount = %d, want default 0", result.TotalCountInfo["danmuCount"])
	}
}

func TestStatisticsServiceGetActualTimeStatisticsInfoRejectsEmptyUserID(t *testing.T) {
	service := NewStatisticsService(&fakeStatisticsRepository{})

	_, err := service.GetActualTimeStatisticsInfo(context.Background(), " ")
	if err == nil {
		t.Fatal("GetActualTimeStatisticsInfo error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
}

func TestStatisticsServiceGetActualTimeStatisticsInfoReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("db error")
	service := NewStatisticsService(&fakeStatisticsRepository{listErr: expectedErr})

	_, err := service.GetActualTimeStatisticsInfo(context.Background(), "1000000001")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}
