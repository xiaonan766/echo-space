package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type StatisticsRepository struct {
	db *gorm.DB
}

type totalStatisticsRow struct {
	PlayCount    int `gorm:"column:play_count"`
	CommentCount int `gorm:"column:comment_count"`
	DanmuCount   int `gorm:"column:danmu_count"`
	LikeCount    int `gorm:"column:like_count"`
	CollectCount int `gorm:"column:collect_count"`
	CoinCount    int `gorm:"column:coin_count"`
}

func NewStatisticsRepository(db *gorm.DB) *StatisticsRepository {
	return &StatisticsRepository{db: db}
}

func (r *StatisticsRepository) ListByUserAndDate(ctx context.Context, userID string, statisticsDate string) ([]domain.StatisticsInfo, error) {
	var list []domain.StatisticsInfo
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND statistics_date = ?", userID, statisticsDate).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.StatisticsInfo{}
	}
	return list, nil
}

func (r *StatisticsRepository) ListByUserTypeAndDateRange(ctx context.Context, userID string, dataType int, startDate string, endDate string) ([]domain.StatisticsInfo, error) {
	var list []domain.StatisticsInfo
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND date_type = ? AND statistics_date >= ? AND statistics_date <= ?", userID, dataType, startDate, endDate).
		Order("statistics_date asc").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.StatisticsInfo{}
	}
	return list, nil
}

func (r *StatisticsRepository) GetTotalStatisticsCountInfo(ctx context.Context, userID string) (map[string]int, error) {
	var row totalStatisticsRow
	err := r.db.WithContext(ctx).
		Table("video_info").
		Select(`
			COALESCE(SUM(play_count), 0) AS play_count,
			COALESCE(SUM(comment_count), 0) AS comment_count,
			COALESCE(SUM(danmu_count), 0) AS danmu_count,
			COALESCE(SUM(like_count), 0) AS like_count,
			COALESCE(SUM(collect_count), 0) AS collect_count,
			COALESCE(SUM(coin_count), 0) AS coin_count
		`).
		Where("user_id = ?", userID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}

	fansCount, err := r.CountFans(ctx, userID)
	if err != nil {
		return nil, err
	}

	return map[string]int{
		"userCount":    int(fansCount),
		"playCount":    row.PlayCount,
		"commentCount": row.CommentCount,
		"danmuCount":   row.DanmuCount,
		"likeCount":    row.LikeCount,
		"collectCount": row.CollectCount,
		"coinCount":    row.CoinCount,
	}, nil
}

func (r *StatisticsRepository) CountFans(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_focus").
		Where("focus_user_id = ?", userID).
		Count(&count).Error
	return count, err
}
