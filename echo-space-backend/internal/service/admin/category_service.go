package admin

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type CategoryService struct {
	categoryRepository *repository.CategoryRepository
}

type SaveCategoryInput struct {
	PCategoryID  int
	CategoryID   int
	CategoryCode string
	CategoryName string
	Icon         string
	Background   string
}

type ChangeCategorySortInput struct {
	PCategoryID int
	CategoryIDs string
}

func NewCategoryService(categoryRepository *repository.CategoryRepository) *CategoryService {
	return &CategoryService{
		categoryRepository: categoryRepository,
	}
}

func (s *CategoryService) LoadCategory(ctx context.Context) ([]domain.CategoryInfo, error) {
	categories, err := s.categoryRepository.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return buildCategoryTree(categories), nil
}

func (s *CategoryService) SaveCategory(ctx context.Context, input SaveCategoryInput) error {
	input.CategoryCode = strings.TrimSpace(input.CategoryCode)
	input.CategoryName = strings.TrimSpace(input.CategoryName)
	input.Icon = strings.TrimSpace(input.Icon)
	input.Background = strings.TrimSpace(input.Background)

	if input.PCategoryID < 0 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if input.CategoryCode == "" {
		return &BusinessError{Info: "\u5206\u7c7b\u7f16\u53f7\u4e0d\u80fd\u4e3a\u7a7a"}
	}
	if input.CategoryName == "" {
		return &BusinessError{Info: "\u5206\u7c7b\u540d\u79f0\u4e0d\u80fd\u4e3a\u7a7a"}
	}
	if input.CategoryID > 0 && input.PCategoryID == input.CategoryID {
		return &BusinessError{Info: "\u5206\u7c7b\u4e0d\u80fd\u4f5c\u4e3a\u81ea\u5df1\u7684\u7236\u7ea7"}
	}

	if err := s.validateParent(ctx, input.PCategoryID); err != nil {
		return err
	}

	existsCount, err := s.categoryRepository.CountByCode(ctx, input.CategoryCode, input.CategoryID)
	if err != nil {
		return err
	}
	if existsCount > 0 {
		return &BusinessError{Info: "\u5206\u7c7b\u7f16\u53f7\u5df2\u5b58\u5728"}
	}

	if input.CategoryID > 0 {
		return s.updateCategory(ctx, input)
	}
	return s.createCategory(ctx, input)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, categoryID int) error {
	if categoryID <= 0 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	if _, err := s.categoryRepository.FindByID(ctx, categoryID); errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "\u5206\u7c7b\u4e0d\u5b58\u5728"}
	} else if err != nil {
		return err
	}

	videoCount, err := s.categoryRepository.CountVideoByCategoryID(ctx, categoryID)
	if err != nil {
		return err
	}
	if videoCount > 0 {
		return &BusinessError{Info: "\u5206\u7c7b\u4e0b\u6709\u89c6\u9891\uff0c\u65e0\u6cd5\u5220\u9664"}
	}

	return s.categoryRepository.DeleteByIDOrParentID(ctx, categoryID)
}

func (s *CategoryService) ChangeSort(ctx context.Context, input ChangeCategorySortInput) error {
	if input.PCategoryID < 0 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	categoryIDs, err := parseCategoryIDs(input.CategoryIDs)
	if err != nil {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	if err := s.validateParent(ctx, input.PCategoryID); err != nil {
		return err
	}

	if err := s.categoryRepository.UpdateSortBatch(ctx, input.PCategoryID, categoryIDs); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &BusinessError{Info: "\u5206\u7c7b\u4e0d\u5b58\u5728"}
		}
		return err
	}
	return nil
}

func (s *CategoryService) createCategory(ctx context.Context, input SaveCategoryInput) error {
	maxSort, err := s.categoryRepository.MaxSortByParent(ctx, input.PCategoryID)
	if err != nil {
		return err
	}

	category := &domain.CategoryInfo{
		PCategoryID:  input.PCategoryID,
		CategoryCode: input.CategoryCode,
		CategoryName: input.CategoryName,
		Icon:         input.Icon,
		Background:   input.Background,
		Sort:         maxSort + 1,
	}
	return s.categoryRepository.Create(ctx, category)
}

func (s *CategoryService) updateCategory(ctx context.Context, input SaveCategoryInput) error {
	category, err := s.categoryRepository.FindByID(ctx, input.CategoryID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "\u5206\u7c7b\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return err
	}

	sort := category.Sort
	if category.PCategoryID != input.PCategoryID {
		maxSort, err := s.categoryRepository.MaxSortByParent(ctx, input.PCategoryID)
		if err != nil {
			return err
		}
		sort = maxSort + 1
	}

	category.PCategoryID = input.PCategoryID
	category.CategoryCode = input.CategoryCode
	category.CategoryName = input.CategoryName
	category.Icon = input.Icon
	category.Background = input.Background
	category.Sort = sort

	return s.categoryRepository.Update(ctx, category)
}

func (s *CategoryService) validateParent(ctx context.Context, pCategoryID int) error {
	if pCategoryID == 0 {
		return nil
	}

	parent, err := s.categoryRepository.FindByID(ctx, pCategoryID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "\u7236\u7ea7\u5206\u7c7b\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return err
	}
	if parent.PCategoryID != 0 {
		return &BusinessError{Info: "\u6682\u53ea\u652f\u6301\u4e8c\u7ea7\u5206\u7c7b"}
	}
	return nil
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

func parseCategoryIDs(categoryIDs string) ([]int, error) {
	parts := strings.Split(categoryIDs, ",")
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, strconv.ErrSyntax
		}

		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			return nil, strconv.ErrSyntax
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, strconv.ErrSyntax
	}
	return ids, nil
}
