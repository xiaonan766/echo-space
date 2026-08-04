package web

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	actionCommentLike  = 0
	actionCommentHate  = 1
	actionVideoLike    = 2
	actionVideoCollect = 3
	actionVideoCoin    = 4
)

type UserActionRepository interface {
	SaveUserAction(ctx context.Context, action domain.UserAction) (domain.UserActionChange, error)
}

type UserActionService struct {
	repository        UserActionRepository
	hotMetricRecorder *VideoHotMetricService
	now               func() time.Time
}

type DoUserActionInput struct {
	UserID      string
	VideoID     string
	ActionType  int
	ActionCount int
	CommentID   int
}

func NewUserActionService(repository UserActionRepository, hotMetricRecorder ...*VideoHotMetricService) *UserActionService {
	service := &UserActionService{
		repository: repository,
		now:        time.Now,
	}
	if len(hotMetricRecorder) > 0 {
		service.hotMetricRecorder = hotMetricRecorder[0]
	}
	return service
}

func (s *UserActionService) DoAction(ctx context.Context, input DoUserActionInput) error {
	input = normalizeDoUserActionInput(input)
	if err := validateDoUserActionInput(input); err != nil {
		return err
	}
	if s == nil || s.repository == nil {
		return errors.New("user action service is not ready")
	}

	change, err := s.repository.SaveUserAction(ctx, domain.UserAction{
		VideoID:     input.VideoID,
		CommentID:   input.CommentID,
		ActionType:  input.ActionType,
		ActionCount: input.ActionCount,
		UserID:      input.UserID,
		ActionTime:  s.currentTime(),
	})
	if err != nil {
		return mapUserActionRepositoryError(err)
	}
	if input.ActionType == actionVideoLike && change.VideoCountDelta != 0 && s.hotMetricRecorder != nil {
		s.hotMetricRecorder.RecordMetric(ctx, input.VideoID, domain.VideoHotMetricEventLike, change.VideoCountDelta)
	}
	return nil
}

func (s *UserActionService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func normalizeDoUserActionInput(input DoUserActionInput) DoUserActionInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.VideoID = strings.TrimSpace(input.VideoID)
	if input.ActionCount == 0 {
		input.ActionCount = 1
	}
	if !isCommentUserAction(input.ActionType) {
		input.CommentID = 0
	}
	return input
}

func validateDoUserActionInput(input DoUserActionInput) error {
	if input.UserID == "" {
		return &BusinessError{Info: "\u8bf7\u5148\u767b\u5f55"}
	}
	if len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID) {
		return &BusinessError{Info: "\u89c6\u9891ID\u4e0d\u6b63\u786e"}
	}
	if !isSupportedUserAction(input.ActionType) {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if input.ActionCount < 1 || input.ActionCount > 2 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if isCommentUserAction(input.ActionType) && input.CommentID <= 0 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	return nil
}

func mapUserActionRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrUserActionVideoNotFound):
		return &BusinessError{Info: "\u89c6\u9891\u4e0d\u5b58\u5728"}
	case errors.Is(err, repository.ErrUserActionCommentNotFound):
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	case errors.Is(err, repository.ErrUserActionSelfCoin):
		return &BusinessError{Info: "UP\u4e3b\u4e0d\u80fd\u7ed9\u81ea\u5df1\u6295\u5e01"}
	case errors.Is(err, repository.ErrUserActionCoinUsed):
		return &BusinessError{Info: "\u5bf9\u672c\u7a3f\u4ef6\u7684\u6295\u5e01\u6b21\u6570\u5df2\u7528\u5b8c"}
	case errors.Is(err, repository.ErrUserActionInsufficientCoin):
		return &BusinessError{Info: "\u786c\u5e01\u4e0d\u8db3"}
	case errors.Is(err, repository.ErrUserActionCoinFailed):
		return &BusinessError{Info: "\u6295\u5e01\u5931\u8d25"}
	default:
		return err
	}
}

func isSupportedUserAction(actionType int) bool {
	return actionType == actionCommentLike ||
		actionType == actionCommentHate ||
		actionType == actionVideoLike ||
		actionType == actionVideoCollect ||
		actionType == actionVideoCoin
}

func isCommentUserAction(actionType int) bool {
	return actionType == actionCommentLike || actionType == actionCommentHate
}
