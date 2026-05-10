package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{
		db: db,
	}
}

func (r *CategoryRepository) ListAll(ctx context.Context) ([]domain.CategoryInfo, error) {
	var categories []domain.CategoryInfo
	err := r.db.WithContext(ctx).
		Order("sort asc").
		Order("category_id asc").
		Find(&categories).Error
	return categories, err
}

func (r *CategoryRepository) FindByID(ctx context.Context, categoryID int) (*domain.CategoryInfo, error) {
	var category domain.CategoryInfo
	err := r.db.WithContext(ctx).
		Where("category_id = ?", categoryID).
		First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) CountByCode(ctx context.Context, categoryCode string, excludeCategoryID int) (int64, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.CategoryInfo{}).
		Where("category_code = ?", categoryCode)
	if excludeCategoryID > 0 {
		query = query.Where("category_id <> ?", excludeCategoryID)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r *CategoryRepository) MaxSortByParent(ctx context.Context, pCategoryID int) (int, error) {
	var maxSort int
	err := r.db.WithContext(ctx).
		Model(&domain.CategoryInfo{}).
		Select("COALESCE(MAX(sort), 0)").
		Where("p_category_id = ?", pCategoryID).
		Scan(&maxSort).Error
	return maxSort, err
}

func (r *CategoryRepository) Create(ctx context.Context, category *domain.CategoryInfo) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *CategoryRepository) Update(ctx context.Context, category *domain.CategoryInfo) error {
	return r.db.WithContext(ctx).
		Model(&domain.CategoryInfo{}).
		Where("category_id = ?", category.CategoryID).
		Updates(map[string]any{
			"p_category_id": category.PCategoryID,
			"category_code": category.CategoryCode,
			"category_name": category.CategoryName,
			"icon":          category.Icon,
			"background":    category.Background,
			"sort":          category.Sort,
		}).Error
}

func (r *CategoryRepository) CountVideoByCategoryID(ctx context.Context, categoryID int) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("video_info").
		Where("category_id = ? OR p_category_id = ?", categoryID, categoryID).
		Count(&count).Error
	return count, err
}

func (r *CategoryRepository) DeleteByIDOrParentID(ctx context.Context, categoryID int) error {
	return r.db.WithContext(ctx).
		Where("category_id = ? OR p_category_id = ?", categoryID, categoryID).
		Delete(&domain.CategoryInfo{}).Error
}

func (r *CategoryRepository) UpdateSortBatch(ctx context.Context, pCategoryID int, categoryIDs []int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index, categoryID := range categoryIDs {
			result := tx.Model(&domain.CategoryInfo{}).
				Where("category_id = ?", categoryID).
				Updates(map[string]any{
					"p_category_id": pCategoryID,
					"sort":          index + 1,
				})
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}
