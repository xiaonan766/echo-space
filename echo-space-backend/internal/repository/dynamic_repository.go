package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type DynamicRepository struct {
	db *gorm.DB
}

type DynamicFeedQuery struct {
	UserID         string
	FocusUserID    string
	PageSize       int
	LastUpdateTime string
	LastVideoID    string
}

func NewDynamicRepository(db *gorm.DB) *DynamicRepository {
	return &DynamicRepository{db: db}
}

func (r *DynamicRepository) ListFollowUsers(ctx context.Context, userID string) ([]domain.DynamicFollowUserItem, error) {
	var list []domain.DynamicFollowUserItem
	err := r.db.WithContext(ctx).
		Table("user_focus uf").
		Select(`
			ui.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(ui.avatar, '') AS avatar,
			COALESCE(ui.person_introduction, '') AS person_introduction,
			DATE_FORMAT(uf.focus_time, '%Y-%m-%d %H:%i:%s') AS focus_time
		`).
		Joins("INNER JOIN user_info ui ON ui.user_id = uf.focus_user_id").
		Where("uf.user_id = ?", userID).
		Order("uf.focus_time desc").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *DynamicRepository) ListFeedByCursor(ctx context.Context, query DynamicFeedQuery) ([]domain.WebVideoItem, error) {
	var list []domain.WebVideoItem
	db := r.db.WithContext(ctx).
		Table("video_info vi").
		Select(webVideoSelectColumns).
		Joins("INNER JOIN user_focus uf ON uf.focus_user_id = vi.user_id AND uf.user_id = ?", query.UserID).
		Joins("LEFT JOIN user_info ui ON vi.user_id = ui.user_id")

	if query.FocusUserID != "" {
		db = db.Where("vi.user_id = ?", query.FocusUserID)
	}
	if query.LastUpdateTime != "" && query.LastVideoID != "" {
		db = db.Where(
			"(DATE_FORMAT(vi.last_update_time, '%Y-%m-%d %H:%i:%s') < ? OR (DATE_FORMAT(vi.last_update_time, '%Y-%m-%d %H:%i:%s') = ? AND vi.video_id < ?))",
			query.LastUpdateTime,
			query.LastUpdateTime,
			query.LastVideoID,
		)
	}

	err := db.
		Order("vi.last_update_time desc, vi.video_id desc").
		Limit(query.PageSize).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}
