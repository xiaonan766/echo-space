package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultUcenterContentPageSize = 15
	maxUcenterContentPageSize     = 100

	ucenterCursorKindComment = "comment"
	ucenterCursorKindDanmu   = "danmu"

	ucenterCursorDirectionNext = "next"
	ucenterCursorDirectionPrev = "prev"
)

type UcenterVideoRepository interface {
	ListUserAllVideo(ctx context.Context, userID string) ([]domain.UcenterAllVideoItem, error)
}

type UcenterInteractRepository interface {
	ListUcenterCommentByCursor(ctx context.Context, query repository.UcenterInteractListQuery) ([]domain.UcenterCommentItem, error)
	ListUcenterDanmuByCursor(ctx context.Context, query repository.UcenterInteractListQuery) ([]domain.UcenterDanmuItem, error)
}

type UcenterContentService struct {
	videoRepository    UcenterVideoRepository
	interactRepository UcenterInteractRepository
}

type UcenterInteractListInput struct {
	UserID   string
	VideoID  string
	Cursor   string
	PageSize int
}

type ucenterCursorPayload struct {
	Kind      string `json:"kind"`
	Direction string `json:"direction"`
	AnchorID  int    `json:"anchorId"`
	VideoID   string `json:"videoId,omitempty"`
}

func NewUcenterContentService(videoRepository UcenterVideoRepository, interactRepository UcenterInteractRepository) *UcenterContentService {
	return &UcenterContentService{
		videoRepository:    videoRepository,
		interactRepository: interactRepository,
	}
}

func (s *UcenterContentService) LoadAllVideo(ctx context.Context, userID string) ([]domain.UcenterAllVideoItem, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, &BusinessError{Info: "请先登录"}
	}
	if s == nil || s.videoRepository == nil {
		return nil, errors.New("ucenter content video repository is not ready")
	}

	list, err := s.videoRepository.ListUserAllVideo(ctx, userID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.UcenterAllVideoItem{}
	}
	return list, nil
}

func (s *UcenterContentService) LoadComment(ctx context.Context, input UcenterInteractListInput) (domain.CursorPaginationResult[domain.UcenterCommentItem], error) {
	input, cursorPayload, hasCursor, err := normalizeUcenterInteractListInput(input, ucenterCursorKindComment)
	if err != nil {
		return domain.CursorPaginationResult[domain.UcenterCommentItem]{}, err
	}
	if s == nil || s.interactRepository == nil {
		return domain.CursorPaginationResult[domain.UcenterCommentItem]{}, errors.New("ucenter content interact repository is not ready")
	}

	list, err := s.interactRepository.ListUcenterCommentByCursor(ctx, repository.UcenterInteractListQuery{
		UserID:    input.UserID,
		VideoID:   input.VideoID,
		CursorID:  cursorPayload.AnchorID,
		Direction: cursorPayload.Direction,
		Limit:     input.PageSize + 1,
	})
	if err != nil {
		return domain.CursorPaginationResult[domain.UcenterCommentItem]{}, err
	}

	return buildUcenterCursorResult(
		list,
		ucenterCursorKindComment,
		input.VideoID,
		input.PageSize,
		cursorPayload.Direction,
		hasCursor,
		func(item domain.UcenterCommentItem) int { return item.CommentID },
	)
}

func (s *UcenterContentService) LoadDanmu(ctx context.Context, input UcenterInteractListInput) (domain.CursorPaginationResult[domain.UcenterDanmuItem], error) {
	input, cursorPayload, hasCursor, err := normalizeUcenterInteractListInput(input, ucenterCursorKindDanmu)
	if err != nil {
		return domain.CursorPaginationResult[domain.UcenterDanmuItem]{}, err
	}
	if s == nil || s.interactRepository == nil {
		return domain.CursorPaginationResult[domain.UcenterDanmuItem]{}, errors.New("ucenter content interact repository is not ready")
	}

	list, err := s.interactRepository.ListUcenterDanmuByCursor(ctx, repository.UcenterInteractListQuery{
		UserID:    input.UserID,
		VideoID:   input.VideoID,
		CursorID:  cursorPayload.AnchorID,
		Direction: cursorPayload.Direction,
		Limit:     input.PageSize + 1,
	})
	if err != nil {
		return domain.CursorPaginationResult[domain.UcenterDanmuItem]{}, err
	}

	return buildUcenterCursorResult(
		list,
		ucenterCursorKindDanmu,
		input.VideoID,
		input.PageSize,
		cursorPayload.Direction,
		hasCursor,
		func(item domain.UcenterDanmuItem) int { return item.DanmuID },
	)
}

func normalizeUcenterInteractListInput(input UcenterInteractListInput, cursorKind string) (UcenterInteractListInput, ucenterCursorPayload, bool, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.VideoID = strings.TrimSpace(input.VideoID)
	input.Cursor = strings.TrimSpace(input.Cursor)
	if input.UserID == "" {
		return input, ucenterCursorPayload{}, false, &BusinessError{Info: "请先登录"}
	}
	if input.VideoID != "" && (len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID)) {
		return input, ucenterCursorPayload{}, false, &BusinessError{Info: "视频ID不正确"}
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultUcenterContentPageSize
	}
	if input.PageSize > maxUcenterContentPageSize {
		input.PageSize = maxUcenterContentPageSize
	}

	cursorPayload, hasCursor, err := decodeUcenterCursor(input.Cursor, cursorKind, input.VideoID)
	if err != nil {
		return input, ucenterCursorPayload{}, false, err
	}
	return input, cursorPayload, hasCursor, nil
}

func decodeUcenterCursor(cursor string, expectedKind string, videoID string) (ucenterCursorPayload, bool, error) {
	if cursor == "" {
		return ucenterCursorPayload{
			Kind:      expectedKind,
			Direction: ucenterCursorDirectionNext,
		}, false, nil
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return ucenterCursorPayload{}, false, &BusinessError{Info: "分页参数错误"}
	}

	var payload ucenterCursorPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return ucenterCursorPayload{}, false, &BusinessError{Info: "分页参数错误"}
	}
	payload.VideoID = strings.TrimSpace(payload.VideoID)
	if payload.Kind != expectedKind {
		return ucenterCursorPayload{}, false, &BusinessError{Info: "分页参数错误"}
	}
	if payload.Direction != ucenterCursorDirectionNext && payload.Direction != ucenterCursorDirectionPrev {
		return ucenterCursorPayload{}, false, &BusinessError{Info: "分页参数错误"}
	}
	if payload.AnchorID <= 0 {
		return ucenterCursorPayload{}, false, &BusinessError{Info: "分页参数错误"}
	}
	if payload.VideoID != videoID {
		return ucenterCursorPayload{}, false, &BusinessError{Info: "分页条件已变化，请重新加载"}
	}
	return payload, true, nil
}

func buildUcenterCursorResult[T any](list []T, cursorKind string, videoID string, pageSize int, direction string, hasCursor bool, getID func(T) int) (domain.CursorPaginationResult[T], error) {
	hasExtra := len(list) > pageSize
	if hasExtra {
		list = list[:pageSize]
	}
	if direction == ucenterCursorDirectionPrev {
		reverseUcenterCursorItems(list)
	}

	hasNext := false
	hasPrev := false
	if len(list) > 0 {
		if direction == ucenterCursorDirectionPrev {
			hasPrev = hasExtra
			hasNext = hasCursor
		} else {
			hasNext = hasExtra
			hasPrev = hasCursor
		}
	}

	nextCursor := ""
	if hasNext && len(list) > 0 {
		cursor, err := encodeUcenterCursor(cursorKind, ucenterCursorDirectionNext, getID(list[len(list)-1]), videoID)
		if err != nil {
			return domain.CursorPaginationResult[T]{}, err
		}
		nextCursor = cursor
	}

	prevCursor := ""
	if hasPrev && len(list) > 0 {
		cursor, err := encodeUcenterCursor(cursorKind, ucenterCursorDirectionPrev, getID(list[0]), videoID)
		if err != nil {
			return domain.CursorPaginationResult[T]{}, err
		}
		prevCursor = cursor
	}

	return domain.NewCursorPaginationResult(list, pageSize, nextCursor, prevCursor, hasNext, hasPrev), nil
}

func encodeUcenterCursor(kind string, direction string, anchorID int, videoID string) (string, error) {
	if anchorID <= 0 {
		return "", nil
	}

	payload := ucenterCursorPayload{
		Kind:      kind,
		Direction: direction,
		AnchorID:  anchorID,
		VideoID:   videoID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payloadBytes), nil
}

func reverseUcenterCursorItems[T any](list []T) {
	for left, right := 0, len(list)-1; left < right; left, right = left+1, right-1 {
		list[left], list[right] = list[right], list[left]
	}
}
