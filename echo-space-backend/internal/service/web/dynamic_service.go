package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultDynamicPageSize = 10
	maxDynamicPageSize     = 30
)

type DynamicRepository interface {
	FindCurrentUserInfo(ctx context.Context, userID string) (*domain.DynamicCurrentUserInfo, error)
	ListFollowUsers(ctx context.Context, userID string) ([]domain.DynamicFollowUserItem, error)
	ListFeedByCursor(ctx context.Context, query repository.DynamicFeedQuery) ([]domain.WebVideoItem, error)
}

type DynamicService struct {
	repository DynamicRepository
}

type LoadDynamicFeedInput struct {
	UserID      string
	FocusUserID string
	Cursor      string
	PageSize    int
}

type dynamicCursorPayload struct {
	FocusUserID    string `json:"focusUserId,omitempty"`
	LastUpdateTime string `json:"lastUpdateTime"`
	LastVideoID    string `json:"lastVideoId"`
}

func NewDynamicService(repository DynamicRepository) *DynamicService {
	return &DynamicService{repository: repository}
}

func (s *DynamicService) LoadCurrentUserInfo(ctx context.Context, userID string) (*domain.DynamicCurrentUserInfo, error) {
	userID = strings.TrimSpace(userID)
	if !validWebUserID(userID) {
		return nil, &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("dynamic service is not ready")
	}

	info, err := s.repository.FindCurrentUserInfo(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "用户不存在"}
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (s *DynamicService) LoadFollowUsers(ctx context.Context, userID string) ([]domain.DynamicFollowUserItem, error) {
	userID = strings.TrimSpace(userID)
	if !validWebUserID(userID) {
		return nil, &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("dynamic service is not ready")
	}

	list, err := s.repository.ListFollowUsers(ctx, userID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.DynamicFollowUserItem{}
	}
	return list, nil
}

func (s *DynamicService) LoadFeed(ctx context.Context, input LoadDynamicFeedInput) (domain.DynamicFeedPage, error) {
	input = normalizeDynamicFeedInput(input)
	if err := validateDynamicFeedInput(input); err != nil {
		return domain.DynamicFeedPage{}, err
	}
	if s == nil || s.repository == nil {
		return domain.DynamicFeedPage{}, errors.New("dynamic service is not ready")
	}

	cursor, err := decodeDynamicCursor(input)
	if err != nil {
		return domain.DynamicFeedPage{}, err
	}

	list, err := s.repository.ListFeedByCursor(ctx, repository.DynamicFeedQuery{
		UserID:         input.UserID,
		FocusUserID:    input.FocusUserID,
		PageSize:       input.PageSize + 1,
		LastUpdateTime: cursor.LastUpdateTime,
		LastVideoID:    cursor.LastVideoID,
	})
	if err != nil {
		return domain.DynamicFeedPage{}, err
	}

	hasMore := len(list) > input.PageSize
	if hasMore {
		list = list[:input.PageSize]
	}
	if list == nil {
		list = []domain.WebVideoItem{}
	}
	fillWebVideoPlayTime(list)

	nextCursor := ""
	if hasMore && len(list) > 0 {
		last := list[len(list)-1]
		nextCursor, err = encodeDynamicCursor(dynamicCursorPayload{
			FocusUserID:    input.FocusUserID,
			LastUpdateTime: last.LastUpdateTime,
			LastVideoID:    last.VideoID,
		})
		if err != nil {
			return domain.DynamicFeedPage{}, err
		}
	}

	return domain.DynamicFeedPage{
		PageSize:   input.PageSize,
		List:       list,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func normalizeDynamicFeedInput(input LoadDynamicFeedInput) LoadDynamicFeedInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.FocusUserID = strings.TrimSpace(input.FocusUserID)
	input.Cursor = strings.TrimSpace(input.Cursor)
	if input.PageSize <= 0 {
		input.PageSize = defaultDynamicPageSize
	}
	if input.PageSize > maxDynamicPageSize {
		input.PageSize = maxDynamicPageSize
	}
	return input
}

func validateDynamicFeedInput(input LoadDynamicFeedInput) error {
	if !validWebUserID(input.UserID) {
		return &BusinessError{Info: "参数错误"}
	}
	if input.FocusUserID != "" && !validWebUserID(input.FocusUserID) {
		return &BusinessError{Info: "参数错误"}
	}
	return nil
}

func decodeDynamicCursor(input LoadDynamicFeedInput) (dynamicCursorPayload, error) {
	if input.Cursor == "" {
		return dynamicCursorPayload{}, nil
	}

	content, err := base64.RawURLEncoding.DecodeString(input.Cursor)
	if err != nil {
		return dynamicCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	var payload dynamicCursorPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return dynamicCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	if payload.FocusUserID != input.FocusUserID ||
		strings.TrimSpace(payload.LastUpdateTime) == "" ||
		len(payload.LastVideoID) != 10 ||
		!isValidPublicVideoID(payload.LastVideoID) {
		return dynamicCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	return payload, nil
}

func encodeDynamicCursor(payload dynamicCursorPayload) (string, error) {
	content, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(content), nil
}
