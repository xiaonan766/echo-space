package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type UserRepository struct {
	db *gorm.DB
}

type UserListQuery struct {
	PageNo        int
	PageSize      int
	NickNameFuzzy string
	Status        *int
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) ListByPage(ctx context.Context, query UserListQuery) ([]domain.UserInfo, int64, error) {
	var totalCount int64
	countDB := r.applyListQuery(r.db.WithContext(ctx).Model(&domain.UserInfo{}), query)
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var users []domain.UserInfo
	offset := (query.PageNo - 1) * query.PageSize
	listDB := r.applyListQuery(r.db.WithContext(ctx).Model(&domain.UserInfo{}), query)
	err := listDB.
		Order("join_time desc").
		Offset(offset).
		Limit(query.PageSize).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, totalCount, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.UserInfo, error) {
	var user domain.UserInfo
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByNickName(ctx context.Context, nickName string) (*domain.UserInfo, error) {
	var user domain.UserInfo
	err := r.db.WithContext(ctx).
		Where("nick_name = ?", nickName).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByUserID(ctx context.Context, userID string) (*domain.UserInfo, error) {
	var user domain.UserInfo
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.UserInfo{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) Create(ctx context.Context, user *domain.UserInfo) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) UpdateStatus(ctx context.Context, userID string, status int) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&domain.UserInfo{}).
		Where("user_id = ?", userID).
		Update("status", status)
	return result.RowsAffected, result.Error
}

func (r *UserRepository) UpdateLoginInfo(ctx context.Context, userID string, loginTime time.Time, loginIP string) error {
	return r.db.WithContext(ctx).
		Model(&domain.UserInfo{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"last_login_time": loginTime,
			"last_login_ip":   loginIP,
		}).Error
}

func (r *UserRepository) CountFocus(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_focus").
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *UserRepository) CountFans(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_focus").
		Where("focus_user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *UserRepository) applyListQuery(db *gorm.DB, query UserListQuery) *gorm.DB {
	if query.NickNameFuzzy != "" {
		db = db.Where("nick_name LIKE ?", "%"+query.NickNameFuzzy+"%")
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	return db
}
