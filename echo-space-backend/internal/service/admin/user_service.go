package admin

import (
	"context"
	"strings"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultPageNo   = 1
	defaultPageSize = 15
	maxPageSize     = 100
)

type UserService struct {
	userRepository *repository.UserRepository
}

type UserListInput struct {
	PageNo        int
	PageSize      int
	NickNameFuzzy string
	Status        *int
}

func NewUserService(userRepository *repository.UserRepository) *UserService {
	return &UserService{
		userRepository: userRepository,
	}
}

func (s *UserService) LoadUser(ctx context.Context, input UserListInput) (domain.PaginationResult[domain.UserInfo], error) {
	input = normalizeUserListInput(input)
	users, totalCount, err := s.userRepository.ListByPage(ctx, repository.UserListQuery{
		PageNo:        input.PageNo,
		PageSize:      input.PageSize,
		NickNameFuzzy: input.NickNameFuzzy,
		Status:        input.Status,
	})
	if err != nil {
		return domain.PaginationResult[domain.UserInfo]{}, err
	}

	return domain.NewPaginationResult(users, totalCount, input.PageNo, input.PageSize), nil
}

func (s *UserService) ChangeStatus(ctx context.Context, userID string, status int) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || (status != 0 && status != 1) {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	rowsAffected, err := s.userRepository.UpdateStatus(ctx, userID, status)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		exists, err := s.userRepository.ExistsByUserID(ctx, userID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		return &BusinessError{Info: "\u7528\u6237\u4e0d\u5b58\u5728"}
	}
	return nil
}

func normalizeUserListInput(input UserListInput) UserListInput {
	if input.PageNo <= 0 {
		input.PageNo = defaultPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultPageSize
	}
	if input.PageSize > maxPageSize {
		input.PageSize = maxPageSize
	}
	input.NickNameFuzzy = strings.TrimSpace(input.NickNameFuzzy)
	return input
}
