package web

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const webUserIDLength = 10

type UhomeRepository interface {
	FindByUserID(ctx context.Context, userID string) (*domain.UserInfo, error)
	FindFocus(ctx context.Context, userID string, focusUserID string) (*domain.UserFocus, error)
	CreateFocus(ctx context.Context, focus *domain.UserFocus) error
}

type UhomeService struct {
	repository UhomeRepository
	now        func() time.Time
}

func NewUhomeService(repository UhomeRepository) *UhomeService {
	return &UhomeService{
		repository: repository,
		now:        time.Now,
	}
}

func (s *UhomeService) FocusUser(ctx context.Context, userID string, focusUserID string) error {
	userID = strings.TrimSpace(userID)
	focusUserID = strings.TrimSpace(focusUserID)
	if !validWebUserID(userID) || !validWebUserID(focusUserID) || userID == focusUserID {
		return &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return errors.New("uhome service is not ready")
	}

	_, err := s.repository.FindFocus(ctx, userID, focusUserID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if _, err = s.repository.FindByUserID(ctx, focusUserID); errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "参数错误"}
	} else if err != nil {
		return err
	}

	return s.repository.CreateFocus(ctx, &domain.UserFocus{
		UserID:      userID,
		FocusUserID: focusUserID,
		FocusTime:   s.currentTime(),
	})
}

func (s *UhomeService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func validWebUserID(userID string) bool {
	return len(userID) == webUserIDLength && isAlphaNumeric(userID)
}
