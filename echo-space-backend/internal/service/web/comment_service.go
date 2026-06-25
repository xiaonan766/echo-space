package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	maxCommentContentLength = 500
	maxCommentImageLength   = 150
	commentPostTimeLayout   = "2006-01-02 15:04:05"

	commentOrderHot    = 0
	commentOrderLatest = 1

	topLevelCommentLimit = 15
	replyCommentLimit    = 10
)

type CommentRepository interface {
	FindCommentTarget(ctx context.Context, videoID string) (*domain.CommentTargetInfo, error)
	FindReplyComment(ctx context.Context, commentID int) (*domain.CommentReplyInfo, error)
	FindTopLevelComment(ctx context.Context, videoID string, commentID int) (*domain.CommentReplyInfo, error)
	CreateComment(ctx context.Context, comment *domain.VideoComment) error
	ListTopComments(ctx context.Context, videoID string) ([]domain.WebCommentItem, error)
	ListTopLevelCommentsByCursor(ctx context.Context, query repository.CommentCursorQuery) ([]domain.WebCommentItem, error)
	ListReplyCommentsByCursor(ctx context.Context, query repository.CommentCursorQuery) ([]domain.WebCommentItem, error)
	CountTopLevelComments(ctx context.Context, videoID string) (int64, error)
	CountReplyComments(ctx context.Context, videoID string, pCommentID int) (int64, error)
	CountRepliesByParentIDs(ctx context.Context, parentIDs []int) (map[int]int, error)
	ListUserCommentActions(ctx context.Context, videoID string, userID string, commentIDs []int) ([]domain.UserActionItem, error)
}

type CommentService struct {
	repository   CommentRepository
	reviewConfig config.CommentReviewConfig
	now          func() time.Time
}

type PostCommentInput struct {
	UserID         string
	NickName       string
	Avatar         string
	Content        string
	ImgPath        string
	VideoID        string
	ReplyCommentID *int
}

type LoadCommentInput struct {
	VideoID    string
	PCommentID int
	OrderType  int
	Cursor     string
	UserID     string
}

type LoadCommentResult struct {
	CommentData    domain.CommentCursorPage `json:"commentData"`
	UserActionList []domain.UserActionItem  `json:"userActionList"`
}

type commentCursorPayload struct {
	VideoID       string `json:"videoId"`
	PCommentID    int    `json:"pCommentId"`
	OrderType     int    `json:"orderType"`
	LastCommentID int    `json:"lastCommentId"`
	LastLikeCount int    `json:"lastLikeCount,omitempty"`
}

func NewCommentService(repository CommentRepository, reviewConfig config.CommentReviewConfig) *CommentService {
	return &CommentService{
		repository:   repository,
		reviewConfig: normalizeCommentReviewConfig(reviewConfig),
		now:          time.Now,
	}
}

func (s *CommentService) LoadComment(ctx context.Context, input LoadCommentInput) (*LoadCommentResult, error) {
	input = normalizeLoadCommentInput(input)
	if err := validateLoadCommentInput(input); err != nil {
		return nil, err
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("comment service is not ready")
	}

	target, err := s.repository.FindCommentTarget(ctx, input.VideoID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "视频不存在"}
	}
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, &BusinessError{Info: "视频不存在"}
	}
	if strings.Contains(target.Interaction, "1") {
		return nil, nil
	}

	cursor, err := decodeCommentCursor(input)
	if err != nil {
		return nil, err
	}
	if input.PCommentID > 0 {
		return s.loadReplyComments(ctx, input, cursor)
	}
	return s.loadTopLevelComments(ctx, input, cursor)
}

func (s *CommentService) loadTopLevelComments(ctx context.Context, input LoadCommentInput, cursor commentCursorPayload) (*LoadCommentResult, error) {
	totalCount, err := s.repository.CountTopLevelComments(ctx, input.VideoID)
	if err != nil {
		return nil, err
	}

	normalComments, err := s.repository.ListTopLevelCommentsByCursor(ctx, repository.CommentCursorQuery{
		VideoID:       input.VideoID,
		OrderType:     input.OrderType,
		Limit:         topLevelCommentLimit + 1,
		LastCommentID: cursor.LastCommentID,
		LastLikeCount: cursor.LastLikeCount,
	})
	if err != nil {
		return nil, err
	}

	hasMore := len(normalComments) > topLevelCommentLimit
	if hasMore {
		normalComments = normalComments[:topLevelCommentLimit]
	}

	list := normalComments
	if input.Cursor == "" {
		topComments, err := s.repository.ListTopComments(ctx, input.VideoID)
		if err != nil {
			return nil, err
		}
		if len(topComments) > 0 {
			list = append(topComments, normalComments...)
		}
	}

	if err := s.fillReplyCount(ctx, list); err != nil {
		return nil, err
	}
	prepareCommentChildren(list)

	nextCursor := ""
	if hasMore && len(normalComments) > 0 {
		last := normalComments[len(normalComments)-1]
		nextCursor, err = encodeCommentCursor(commentCursorPayload{
			VideoID:       input.VideoID,
			PCommentID:    0,
			OrderType:     input.OrderType,
			LastCommentID: last.CommentID,
			LastLikeCount: last.LikeCount,
		})
		if err != nil {
			return nil, err
		}
	}

	actions, err := s.loadUserActions(ctx, input.VideoID, input.UserID, list)
	if err != nil {
		return nil, err
	}

	return &LoadCommentResult{
		CommentData: domain.CommentCursorPage{
			TotalCount: totalCount,
			PageSize:   topLevelCommentLimit,
			List:       list,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
		UserActionList: actions,
	}, nil
}

func (s *CommentService) loadReplyComments(ctx context.Context, input LoadCommentInput, cursor commentCursorPayload) (*LoadCommentResult, error) {
	if _, err := s.repository.FindTopLevelComment(ctx, input.VideoID, input.PCommentID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "参数错误"}
	} else if err != nil {
		return nil, err
	}

	totalCount, err := s.repository.CountReplyComments(ctx, input.VideoID, input.PCommentID)
	if err != nil {
		return nil, err
	}
	list, err := s.repository.ListReplyCommentsByCursor(ctx, repository.CommentCursorQuery{
		VideoID:       input.VideoID,
		PCommentID:    input.PCommentID,
		Limit:         replyCommentLimit + 1,
		LastCommentID: cursor.LastCommentID,
	})
	if err != nil {
		return nil, err
	}

	hasMore := len(list) > replyCommentLimit
	if hasMore {
		list = list[:replyCommentLimit]
	}
	prepareCommentChildren(list)

	nextCursor := ""
	if hasMore && len(list) > 0 {
		last := list[len(list)-1]
		nextCursor, err = encodeCommentCursor(commentCursorPayload{
			VideoID:       input.VideoID,
			PCommentID:    input.PCommentID,
			OrderType:     commentOrderHot,
			LastCommentID: last.CommentID,
		})
		if err != nil {
			return nil, err
		}
	}

	actions, err := s.loadUserActions(ctx, input.VideoID, input.UserID, list)
	if err != nil {
		return nil, err
	}

	return &LoadCommentResult{
		CommentData: domain.CommentCursorPage{
			TotalCount: totalCount,
			PageSize:   replyCommentLimit,
			List:       list,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
		UserActionList: actions,
	}, nil
}

func (s *CommentService) PostComment(ctx context.Context, input PostCommentInput) (*domain.WebCommentItem, error) {
	input = normalizePostCommentInput(input)
	if err := validatePostCommentInput(input); err != nil {
		return nil, err
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("comment service is not ready")
	}
	if err := s.reviewContent(input.Content); err != nil {
		return nil, err
	}

	target, err := s.repository.FindCommentTarget(ctx, input.VideoID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "视频不存在"}
	}
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, &BusinessError{Info: "视频不存在"}
	}
	if strings.Contains(target.Interaction, "1") {
		return nil, &BusinessError{Info: "UP主已关闭评论区"}
	}

	comment := &domain.VideoComment{
		PCommentID:  0,
		VideoID:     input.VideoID,
		VideoUserID: target.VideoUserID,
		Content:     input.Content,
		ImgPath:     input.ImgPath,
		UserID:      input.UserID,
		TopType:     0,
		PostTime:    s.currentTime(),
		LikeCount:   0,
		HateCount:   0,
	}
	result := domain.WebCommentItem{
		VideoID:     input.VideoID,
		VideoUserID: target.VideoUserID,
		UserID:      input.UserID,
		Avatar:      input.Avatar,
		NickName:    input.NickName,
		Content:     input.Content,
		ImgPath:     input.ImgPath,
		TopType:     0,
		LikeCount:   0,
		HateCount:   0,
		Children:    []domain.WebCommentItem{},
	}

	if input.ReplyCommentID != nil {
		replyComment, err := s.repository.FindReplyComment(ctx, *input.ReplyCommentID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &BusinessError{Info: "参数错误"}
		}
		if err != nil {
			return nil, err
		}
		if replyComment == nil || replyComment.VideoID != input.VideoID {
			return nil, &BusinessError{Info: "参数错误"}
		}

		if replyComment.PCommentID == 0 {
			comment.PCommentID = replyComment.CommentID
		} else {
			comment.PCommentID = replyComment.PCommentID
			comment.ReplyUserID = replyComment.UserID
			result.ReplyUserID = replyComment.UserID
			result.ReplyNickName = replyComment.NickName
			result.ReplyAvatar = replyComment.Avatar
		}
	}

	if err := s.repository.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	result.CommentID = comment.CommentID
	result.PCommentID = comment.PCommentID
	result.PostTime = comment.PostTime.Format(commentPostTimeLayout)
	return &result, nil
}

func (s *CommentService) fillReplyCount(ctx context.Context, list []domain.WebCommentItem) error {
	parentIDs := make([]int, 0, len(list))
	for _, item := range list {
		if item.PCommentID == 0 {
			parentIDs = append(parentIDs, item.CommentID)
		}
	}
	replyCountMap, err := s.repository.CountRepliesByParentIDs(ctx, parentIDs)
	if err != nil {
		return err
	}
	for index := range list {
		list[index].ReplyCount = replyCountMap[list[index].CommentID]
	}
	return nil
}

func (s *CommentService) loadUserActions(ctx context.Context, videoID string, userID string, list []domain.WebCommentItem) ([]domain.UserActionItem, error) {
	commentIDs := make([]int, 0, len(list))
	for _, item := range list {
		commentIDs = append(commentIDs, item.CommentID)
	}
	return s.repository.ListUserCommentActions(ctx, videoID, userID, commentIDs)
}

func (s *CommentService) reviewContent(content string) error {
	config := s.reviewConfig
	if !config.Enabled {
		return nil
	}

	content = strings.ToLower(content)
	for _, word := range config.SensitiveWords {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			continue
		}
		if strings.Contains(content, word) {
			return &BusinessError{Info: config.RejectMessage}
		}
	}
	return nil
}

func (s *CommentService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func normalizeLoadCommentInput(input LoadCommentInput) LoadCommentInput {
	input.VideoID = strings.TrimSpace(input.VideoID)
	input.Cursor = strings.TrimSpace(input.Cursor)
	input.UserID = strings.TrimSpace(input.UserID)
	if input.PCommentID > 0 {
		input.OrderType = commentOrderHot
	}
	return input
}

func validateLoadCommentInput(input LoadCommentInput) error {
	if len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID) {
		return &BusinessError{Info: "视频ID不正确"}
	}
	if input.PCommentID < 0 {
		return &BusinessError{Info: "参数错误"}
	}
	if input.OrderType != commentOrderHot && input.OrderType != commentOrderLatest {
		return &BusinessError{Info: "参数错误"}
	}
	return nil
}

func normalizePostCommentInput(input PostCommentInput) PostCommentInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.NickName = strings.TrimSpace(input.NickName)
	input.Avatar = strings.TrimSpace(input.Avatar)
	input.Content = strings.TrimSpace(input.Content)
	input.ImgPath = strings.TrimSpace(input.ImgPath)
	input.VideoID = strings.TrimSpace(input.VideoID)
	return input
}

func validatePostCommentInput(input PostCommentInput) error {
	if input.UserID == "" {
		return &BusinessError{Info: "请先登录"}
	}
	if input.Content == "" || utf8.RuneCountInString(input.Content) > maxCommentContentLength {
		return &BusinessError{Info: "评论内容不能为空且不能超过500个字"}
	}
	if utf8.RuneCountInString(input.ImgPath) > maxCommentImageLength {
		return &BusinessError{Info: "评论图片不正确"}
	}
	if len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID) {
		return &BusinessError{Info: "视频ID不正确"}
	}
	if input.ReplyCommentID != nil && *input.ReplyCommentID <= 0 {
		return &BusinessError{Info: "参数错误"}
	}
	return nil
}

func normalizeCommentReviewConfig(reviewConfig config.CommentReviewConfig) config.CommentReviewConfig {
	isZeroConfig := !reviewConfig.Enabled && strings.TrimSpace(reviewConfig.RejectMessage) == "" && len(reviewConfig.SensitiveWords) == 0
	if isZeroConfig {
		reviewConfig.Enabled = true
	}
	if strings.TrimSpace(reviewConfig.RejectMessage) == "" {
		reviewConfig.RejectMessage = "评论包含敏感内容，请修改后再发布"
	}
	return reviewConfig
}

func decodeCommentCursor(input LoadCommentInput) (commentCursorPayload, error) {
	if input.Cursor == "" {
		return commentCursorPayload{}, nil
	}

	content, err := base64.RawURLEncoding.DecodeString(input.Cursor)
	if err != nil {
		return commentCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	var payload commentCursorPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return commentCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	if payload.VideoID != input.VideoID || payload.PCommentID != input.PCommentID || payload.OrderType != input.OrderType || payload.LastCommentID <= 0 {
		return commentCursorPayload{}, &BusinessError{Info: "参数错误"}
	}
	return payload, nil
}

func encodeCommentCursor(payload commentCursorPayload) (string, error) {
	content, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(content), nil
}

func prepareCommentChildren(list []domain.WebCommentItem) {
	for index := range list {
		if list[index].Children == nil {
			list[index].Children = []domain.WebCommentItem{}
		}
	}
}
