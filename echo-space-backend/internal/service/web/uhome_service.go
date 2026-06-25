package web

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const webUserIDLength = 10
const updateNickNameCoin = 5

type UhomeRepository interface {
	FindByUserID(ctx context.Context, userID string) (*domain.UserInfo, error)
	FindByNickName(ctx context.Context, nickName string) (*domain.UserInfo, error)
	UpdateUserInfo(ctx context.Context, userID string, userInfo domain.UserInfo, spendCoin int) (*domain.UserInfo, error)
	FindFocus(ctx context.Context, userID string, focusUserID string) (*domain.UserFocus, error)
	CreateFocus(ctx context.Context, focus *domain.UserFocus) error
	CountFocus(ctx context.Context, userID string) (int64, error)
	CountFans(ctx context.Context, userID string) (int64, error)
	SumUserVideoCount(ctx context.Context, userID string) (domain.UserVideoCountInfo, error)
}

type UhomeService struct {
	repository UhomeRepository
	now        func() time.Time
}

type UpdateUserInfoInput struct {
	UserID             string
	Avatar             string
	NickName           string
	Sex                int
	Birthday           string
	School             string
	PersonIntroduction string
	NoticeInfo         string
}

func NewUhomeService(repository UhomeRepository) *UhomeService {
	return &UhomeService{
		repository: repository,
		now:        time.Now,
	}
}

func (s *UhomeService) UpdateUserInfo(ctx context.Context, currentUserID string, input UpdateUserInfoInput) (*domain.UserInfo, error) {
	currentUserID = strings.TrimSpace(currentUserID)
	input = normalizeUpdateUserInfoInput(input)
	if err := validateUpdateUserInfoInput(currentUserID, input); err != nil {
		return nil, err
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("uhome service is not ready")
	}

	currentUser, err := s.repository.FindByUserID(ctx, input.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "\u7528\u6237\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return nil, err
	}

	spendCoin := 0
	if currentUser.NickName != input.NickName {
		if currentUser.CurrentCoinCount < updateNickNameCoin {
			return nil, &BusinessError{Info: "\u786c\u5e01\u4e0d\u8db3\uff0c\u65e0\u6cd5\u4fee\u6539\u6635\u79f0"}
		}
		nickNameUser, err := s.repository.FindByNickName(ctx, input.NickName)
		if err == nil && nickNameUser.UserID != input.UserID {
			return nil, &BusinessError{Info: "\u6635\u79f0\u5df2\u5b58\u5728"}
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		spendCoin = updateNickNameCoin
	}

	updatedUser, err := s.repository.UpdateUserInfo(ctx, input.UserID, domain.UserInfo{
		Avatar:      input.Avatar,
		NickName:    input.NickName,
		Sex:         input.Sex,
		Birthday:    input.Birthday,
		School:      input.School,
		PersonIntro: input.PersonIntroduction,
		NoticeInfo:  input.NoticeInfo,
	}, spendCoin)
	if err != nil {
		return nil, mapUpdateUserInfoError(err)
	}
	return updatedUser, nil
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

func (s *UhomeService) GetUserInfo(ctx context.Context, currentUserID string, userID string) (*domain.UserHomeInfo, error) {
	currentUserID = strings.TrimSpace(currentUserID)
	userID = strings.TrimSpace(userID)
	if !validWebUserID(userID) || (currentUserID != "" && !validWebUserID(currentUserID)) {
		return nil, &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("uhome service is not ready")
	}

	userInfo, err := s.repository.FindByUserID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &NotFoundError{Info: "用户不存在"}
	}
	if err != nil {
		return nil, err
	}

	videoCountInfo, err := s.repository.SumUserVideoCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	fansCount, err := s.repository.CountFans(ctx, userID)
	if err != nil {
		return nil, err
	}
	focusCount, err := s.repository.CountFocus(ctx, userID)
	if err != nil {
		return nil, err
	}

	haveFocus := false
	if currentUserID != "" {
		_, err := s.repository.FindFocus(ctx, currentUserID, userID)
		if err == nil {
			haveFocus = true
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	return &domain.UserHomeInfo{
		UserID:             userInfo.UserID,
		NickName:           userInfo.NickName,
		Sex:                userInfo.Sex,
		Birthday:           userInfo.Birthday,
		School:             userInfo.School,
		PersonIntroduction: userInfo.PersonIntro,
		NoticeInfo:         userInfo.NoticeInfo,
		Theme:              userInfo.Theme,
		Avatar:             userInfo.Avatar,
		PlayCount:          videoCountInfo.PlayCount,
		LikeCount:          videoCountInfo.LikeCount,
		FansCount:          int(fansCount),
		FocusCount:         int(focusCount),
		HaveFocus:          haveFocus,
	}, nil
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

func normalizeUpdateUserInfoInput(input UpdateUserInfoInput) UpdateUserInfoInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.Avatar = strings.TrimSpace(input.Avatar)
	input.NickName = strings.TrimSpace(input.NickName)
	input.Birthday = strings.TrimSpace(input.Birthday)
	input.School = strings.TrimSpace(input.School)
	input.PersonIntroduction = strings.TrimSpace(input.PersonIntroduction)
	input.NoticeInfo = strings.TrimSpace(input.NoticeInfo)
	return input
}

func validateUpdateUserInfoInput(currentUserID string, input UpdateUserInfoInput) error {
	if !validWebUserID(currentUserID) || !validWebUserID(input.UserID) || currentUserID != input.UserID {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if input.Avatar == "" || utf8.RuneCountInString(input.Avatar) > 100 {
		return &BusinessError{Info: "\u8bf7\u4e0a\u4f20\u5934\u50cf"}
	}
	if input.NickName == "" || utf8.RuneCountInString(input.NickName) > 20 {
		return &BusinessError{Info: "\u8bf7\u8f93\u5165\u6635\u79f0"}
	}
	if input.Sex != 0 && input.Sex != 1 && input.Sex != 2 {
		return &BusinessError{Info: "\u8bf7\u9009\u62e9\u6027\u522b"}
	}
	if utf8.RuneCountInString(input.Birthday) > 10 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if utf8.RuneCountInString(input.School) > 150 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if utf8.RuneCountInString(input.PersonIntroduction) > 80 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if utf8.RuneCountInString(input.NoticeInfo) > 300 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	return nil
}

func mapUpdateUserInfoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrUpdateUserInfoNotFound):
		return &BusinessError{Info: "\u7528\u6237\u4e0d\u5b58\u5728"}
	case errors.Is(err, repository.ErrUpdateUserInfoInsufficientCoin):
		return &BusinessError{Info: "\u786c\u5e01\u4e0d\u8db3\uff0c\u65e0\u6cd5\u4fee\u6539\u6635\u79f0"}
	default:
		return err
	}
}
