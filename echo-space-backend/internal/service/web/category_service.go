package web

import (
	"context"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type CategoryService struct {
	categoryRepository *repository.CategoryRepository
}

func NewCategoryService(categoryRepository *repository.CategoryRepository) *CategoryService {
	return &CategoryService{
		categoryRepository: categoryRepository,
	}
}

func (s *CategoryService) LoadAllCategory(ctx context.Context) ([]domain.CategoryInfo, error) {
	categories, err := s.categoryRepository.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return buildCategoryTree(categories), nil
}

func buildCategoryTree(categories []domain.CategoryInfo) []domain.CategoryInfo {
	grouped := make(map[int][]domain.CategoryInfo)
	for _, category := range categories {
		category.Children = []domain.CategoryInfo{}
		grouped[category.PCategoryID] = append(grouped[category.PCategoryID], category)
	}
	return buildCategoryChildren(0, grouped)
}

func buildCategoryChildren(parentID int, grouped map[int][]domain.CategoryInfo) []domain.CategoryInfo {
	children := grouped[parentID]
	if len(children) == 0 {
		return []domain.CategoryInfo{}
	}

	result := make([]domain.CategoryInfo, 0, len(children))
	for _, child := range children {
		child.Children = buildCategoryChildren(child.CategoryID, grouped)
		result = append(result, child)
	}
	return result
}
