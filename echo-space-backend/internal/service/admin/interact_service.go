package admin

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type InteractService struct {
	interactRepository *repository.InteractRepository
}

type InteractListInput struct {
	PageNo         int
	PageSize       int
	VideoNameFuzzy string
}

func NewInteractService(interactRepository *repository.InteractRepository) *InteractService {
	return &InteractService{
		interactRepository: interactRepository,
	}
}

func (s *InteractService) LoadComment(ctx context.Context, input InteractListInput) (domain.PaginationResult[domain.AdminCommentItem], error) {
	input = normalizeInteractListInput(input)
	comments, totalCount, err := s.interactRepository.ListCommentByPage(ctx, repository.InteractListQuery{
		PageNo:         input.PageNo,
		PageSize:       input.PageSize,
		VideoNameFuzzy: input.VideoNameFuzzy,
	})
	if err != nil {
		return domain.PaginationResult[domain.AdminCommentItem]{}, err
	}
	return domain.NewPaginationResult(comments, totalCount, input.PageNo, input.PageSize), nil
}

func (s *InteractService) DeleteComment(ctx context.Context, commentID int) error {
	if commentID <= 0 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	comment, err := s.interactRepository.FindCommentDeleteInfo(ctx, commentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "\u8bc4\u8bba\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return err
	}

	exists, err := s.interactRepository.VideoExists(ctx, comment.VideoID)
	if err != nil {
		return err
	}
	if !exists {
		return &BusinessError{Info: "\u89c6\u9891\u4e0d\u5b58\u5728"}
	}

	if err := s.interactRepository.DeleteComment(ctx, *comment); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &BusinessError{Info: "\u8bc4\u8bba\u4e0d\u5b58\u5728"}
		}
		return err
	}
	return nil
}

func (s *InteractService) LoadDanmu(ctx context.Context, input InteractListInput) (domain.PaginationResult[domain.AdminDanmuItem], error) {
	input = normalizeInteractListInput(input)
	danmuList, totalCount, err := s.interactRepository.ListDanmuByPage(ctx, repository.InteractListQuery{
		PageNo:         input.PageNo,
		PageSize:       input.PageSize,
		VideoNameFuzzy: input.VideoNameFuzzy,
	})
	if err != nil {
		return domain.PaginationResult[domain.AdminDanmuItem]{}, err
	}
	return domain.NewPaginationResult(danmuList, totalCount, input.PageNo, input.PageSize), nil
}

func (s *InteractService) DeleteDanmu(ctx context.Context, danmuID int) error {
	if danmuID <= 0 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	danmu, err := s.interactRepository.FindDanmuDeleteInfo(ctx, danmuID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "\u5f39\u5e55\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return err
	}

	exists, err := s.interactRepository.VideoExists(ctx, danmu.VideoID)
	if err != nil {
		return err
	}
	if !exists {
		return &BusinessError{Info: "\u89c6\u9891\u4e0d\u5b58\u5728"}
	}

	if err := s.interactRepository.DeleteDanmu(ctx, *danmu); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &BusinessError{Info: "\u5f39\u5e55\u4e0d\u5b58\u5728"}
		}
		return err
	}
	return nil
}

func normalizeInteractListInput(input InteractListInput) InteractListInput {
	if input.PageNo <= 0 {
		input.PageNo = defaultPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultPageSize
	}
	if input.PageSize > maxPageSize {
		input.PageSize = maxPageSize
	}
	input.VideoNameFuzzy = strings.TrimSpace(input.VideoNameFuzzy)
	return input
}
