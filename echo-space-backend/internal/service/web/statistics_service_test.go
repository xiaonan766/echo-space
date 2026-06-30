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
	weekList       []domain.StatisticsInfo
	total          map[string]int
	listErr        error
	weekErr        error
	totalErr       error
	userID         string
	statisticsDate string
	weekUserID     string
	weekDataType   int
	weekStartDate  string
	weekEndDate    string
}

func (r *fakeStatisticsRepository) ListByUserAndDate(ctx context.Context, userID string, statisticsDate string) ([]domain.StatisticsInfo, error) {
	r.userID = userID
	r.statisticsDate = statisticsDate
	return r.list, r.listErr
}

func (r *fakeStatisticsRepository) ListByUserTypeAndDateRange(ctx context.Context, userID string, dataType int, startDate string, endDate string) ([]domain.StatisticsInfo, error) {
	r.weekUserID = userID
	r.weekDataType = dataType
	r.weekStartDate = startDate
	r.weekEndDate = endDate
	return r.weekList, r.weekErr
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

func TestStatisticsServiceGetWeekStatisticsInfo(t *testing.T) {
	repository := &fakeStatisticsRepository{
		weekList: []domain.StatisticsInfo{
			{StatisticsDate: "2026-06-24", UserID: "1000000001", DateType: statisticsTypePlay, StatisticsCount: 12},
			{StatisticsDate: "2026-06-29", UserID: "1000000001", DateType: statisticsTypePlay, StatisticsCount: 8},
		},
	}
	service := NewStatisticsService(repository)
	service.now = func() time.Time {
		return time.Date(2026, 6, 30, 12, 0, 0, 0, time.Local)
	}

	result, err := service.GetWeekStatisticsInfo(context.Background(), "1000000001", statisticsTypePlay)
	if err != nil {
		t.Fatalf("GetWeekStatisticsInfo error = %v", err)
	}
	if repository.weekUserID != "1000000001" {
		t.Fatalf("repository userID = %s, want %s", repository.weekUserID, "1000000001")
	}
	if repository.weekDataType != statisticsTypePlay {
		t.Fatalf("repository dataType = %d, want %d", repository.weekDataType, statisticsTypePlay)
	}
	if repository.weekStartDate != "2026-06-23" || repository.weekEndDate != "2026-06-29" {
		t.Fatalf("date range = %s..%s, want 2026-06-23..2026-06-29", repository.weekStartDate, repository.weekEndDate)
	}
	if len(result) != 7 {
		t.Fatalf("len(result) = %d, want 7", len(result))
	}

	expectedDates := []string{
		"2026-06-23",
		"2026-06-24",
		"2026-06-25",
		"2026-06-26",
		"2026-06-27",
		"2026-06-28",
		"2026-06-29",
	}
	expectedCounts := []int{0, 12, 0, 0, 0, 0, 8}
	for index := range expectedDates {
		if result[index].StatisticsDate != expectedDates[index] {
			t.Fatalf("result[%d].StatisticsDate = %s, want %s", index, result[index].StatisticsDate, expectedDates[index])
		}
		if result[index].StatisticsCount != expectedCounts[index] {
			t.Fatalf("result[%d].StatisticsCount = %d, want %d", index, result[index].StatisticsCount, expectedCounts[index])
		}
		if result[index].DateType != statisticsTypePlay {
			t.Fatalf("result[%d].DateType = %d, want %d", index, result[index].DateType, statisticsTypePlay)
		}
	}
}

func TestStatisticsServiceGetWeekStatisticsInfoRejectsInvalidDataType(t *testing.T) {
	service := NewStatisticsService(&fakeStatisticsRepository{})

	_, err := service.GetWeekStatisticsInfo(context.Background(), "1000000001", 99)
	if err == nil {
		t.Fatal("GetWeekStatisticsInfo error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
}

func TestStatisticsServiceGetWeekStatisticsInfoReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("db error")
	service := NewStatisticsService(&fakeStatisticsRepository{weekErr: expectedErr})

	_, err := service.GetWeekStatisticsInfo(context.Background(), "1000000001", statisticsTypePlay)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}
