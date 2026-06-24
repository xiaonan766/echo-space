package web

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const (
	maxCommentContentLength = 500
	maxCommentImageLength   = 150
	commentPostTimeLayout   = "2006-01-02 15:04:05"
)

type CommentRepository interface {
	FindCommentTarget(ctx context.Context, videoID string) (*domain.CommentTargetInfo, error)
	FindReplyComment(ctx context.Context, commentID int) (*domain.CommentReplyInfo, error)
	CreateComment(ctx context.Context, comment *domain.VideoComment) error
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

func NewCommentService(repository CommentRepository, reviewConfig config.CommentReviewConfig) *CommentService {
	return &CommentService{
		repository:   repository,
		reviewConfig: normalizeCommentReviewConfig(reviewConfig),
		now:          time.Now,
	}
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
