package web

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

func TestValidatePostCommentInput(t *testing.T) {
	valid := PostCommentInput{
		UserID:   "10001",
		NickName: "测试用户",
		Content:  "这是一条评论",
		VideoID:  "Abc123Def4",
	}
	if err := validatePostCommentInput(valid); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}

	invalidReplyID := -1
	tests := []struct {
		name   string
		mutate func(*PostCommentInput)
	}{
		{name: "empty content", mutate: func(input *PostCommentInput) { input.Content = "" }},
		{name: "long content", mutate: func(input *PostCommentInput) { input.Content = strings.Repeat("字", 501) }},
		{name: "long image path", mutate: func(input *PostCommentInput) { input.ImgPath = strings.Repeat("a", 151) }},
		{name: "invalid video id", mutate: func(input *PostCommentInput) { input.VideoID = "bad" }},
		{name: "invalid reply id", mutate: func(input *PostCommentInput) { input.ReplyCommentID = &invalidReplyID }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if err := validatePostCommentInput(input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestPostCommentRejectsSensitiveWordBeforeWrite(t *testing.T) {
	repo := &fakeCommentRepository{
		target: &domain.CommentTargetInfo{VideoID: "Abc123Def4", VideoUserID: "20001"},
	}
	service := NewCommentService(repo, config.CommentReviewConfig{
		Enabled:        true,
		RejectMessage:  "评论包含敏感内容，请修改后再发布",
		SensitiveWords: []string{"广告"},
	})

	_, err := service.PostComment(context.Background(), PostCommentInput{
		UserID:  "10001",
		Content: "这里有广告",
		VideoID: "Abc123Def4",
	})
	if err == nil {
		t.Fatal("expected sensitive word error")
	}
	businessError, ok := IsBusinessError(err)
	if !ok || businessError.Info != "评论包含敏感内容，请修改后再发布" {
		t.Fatalf("error = %#v, want sensitive business error", err)
	}
	if repo.createCount != 0 {
		t.Fatalf("createCount = %d, want 0", repo.createCount)
	}
}

func TestPostCommentReviewIsCaseInsensitive(t *testing.T) {
	service := NewCommentService(&fakeCommentRepository{}, config.CommentReviewConfig{
		Enabled:        true,
		RejectMessage:  "blocked",
		SensitiveWords: []string{"vx"},
	})

	err := service.reviewContent("加 VX 联系")
	if err == nil {
		t.Fatal("expected case-insensitive sensitive word error")
	}
}

func TestPostCommentVideoNotFound(t *testing.T) {
	repo := &fakeCommentRepository{targetErr: gorm.ErrRecordNotFound}
	service := NewCommentService(repo, config.CommentReviewConfig{})

	_, err := service.PostComment(context.Background(), PostCommentInput{
		UserID:  "10001",
		Content: "正常评论",
		VideoID: "Abc123Def4",
	})
	if err == nil {
		t.Fatal("expected video not found error")
	}
	businessError, ok := IsBusinessError(err)
	if !ok || businessError.Info != "视频不存在" {
		t.Fatalf("error = %#v, want video not found business error", err)
	}
}

func TestPostCommentRejectsClosedCommentArea(t *testing.T) {
	repo := &fakeCommentRepository{
		target: &domain.CommentTargetInfo{VideoID: "Abc123Def4", VideoUserID: "20001", Interaction: "1"},
	}
	service := NewCommentService(repo, config.CommentReviewConfig{})

	_, err := service.PostComment(context.Background(), PostCommentInput{
		UserID:  "10001",
		Content: "正常评论",
		VideoID: "Abc123Def4",
	})
	if err == nil {
		t.Fatal("expected closed comment area error")
	}
	businessError, ok := IsBusinessError(err)
	if !ok || businessError.Info != "UP主已关闭评论区" {
		t.Fatalf("error = %#v, want closed comment business error", err)
	}
}

func TestPostCommentCreatesTopLevelComment(t *testing.T) {
	repo := &fakeCommentRepository{
		target: &domain.CommentTargetInfo{VideoID: "Abc123Def4", VideoUserID: "20001"},
	}
	service := NewCommentService(repo, config.CommentReviewConfig{})
	service.now = func() time.Time {
		return time.Date(2026, 6, 24, 14, 30, 0, 0, time.Local)
	}

	result, err := service.PostComment(context.Background(), PostCommentInput{
		UserID:   "10001",
		NickName: "测试用户",
		Avatar:   "avatar.png",
		Content:  "正常评论",
		ImgPath:  "cover/comment.png",
		VideoID:  "Abc123Def4",
	})
	if err != nil {
		t.Fatalf("post comment returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected comment to be created")
	}
	if repo.created.PCommentID != 0 || repo.topLevelCreateCount != 1 {
		t.Fatalf("created pCommentId=%d topLevelCreateCount=%d, want 0 and 1", repo.created.PCommentID, repo.topLevelCreateCount)
	}
	if result.CommentID != 100 || result.PCommentID != 0 || result.NickName != "测试用户" || result.PostTime != "2026-06-24 14:30:00" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestPostCommentCreatesNestedReply(t *testing.T) {
	replyID := 8
	repo := &fakeCommentRepository{
		target: &domain.CommentTargetInfo{VideoID: "Abc123Def4", VideoUserID: "20001"},
		reply: &domain.CommentReplyInfo{
			CommentID:  8,
			PCommentID: 7,
			VideoID:    "Abc123Def4",
			UserID:     "30001",
			NickName:   "被回复用户",
			Avatar:     "reply-avatar.png",
		},
	}
	service := NewCommentService(repo, config.CommentReviewConfig{})

	result, err := service.PostComment(context.Background(), PostCommentInput{
		UserID:         "10001",
		NickName:       "测试用户",
		Content:        "回复内容",
		VideoID:        "Abc123Def4",
		ReplyCommentID: &replyID,
	})
	if err != nil {
		t.Fatalf("post comment returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected comment to be created")
	}
	if repo.created.PCommentID != 7 || repo.created.ReplyUserID != "30001" {
		t.Fatalf("created comment = %#v, want parent 7 and reply user 30001", repo.created)
	}
	if repo.topLevelCreateCount != 0 {
		t.Fatalf("topLevelCreateCount = %d, want 0", repo.topLevelCreateCount)
	}
	if result.PCommentID != 7 || result.ReplyUserID != "30001" || result.ReplyNickName != "被回复用户" || result.ReplyAvatar != "reply-avatar.png" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestPostCommentRejectsReplyFromOtherVideo(t *testing.T) {
	replyID := 8
	repo := &fakeCommentRepository{
		target: &domain.CommentTargetInfo{VideoID: "Abc123Def4", VideoUserID: "20001"},
		reply:  &domain.CommentReplyInfo{CommentID: 8, PCommentID: 0, VideoID: "Other12345", UserID: "30001"},
	}
	service := NewCommentService(repo, config.CommentReviewConfig{})

	_, err := service.PostComment(context.Background(), PostCommentInput{
		UserID:         "10001",
		Content:        "回复内容",
		VideoID:        "Abc123Def4",
		ReplyCommentID: &replyID,
	})
	if err == nil {
		t.Fatal("expected reply validation error")
	}
	businessError, ok := IsBusinessError(err)
	if !ok || businessError.Info != "参数错误" {
		t.Fatalf("error = %#v, want param business error", err)
	}
}

type fakeCommentRepository struct {
	target *domain.CommentTargetInfo
	reply  *domain.CommentReplyInfo

	targetErr error
	replyErr  error
	createErr error

	created             *domain.VideoComment
	createCount         int
	topLevelCreateCount int
}

func (r *fakeCommentRepository) FindCommentTarget(ctx context.Context, videoID string) (*domain.CommentTargetInfo, error) {
	if r.targetErr != nil {
		return nil, r.targetErr
	}
	return r.target, nil
}

func (r *fakeCommentRepository) FindReplyComment(ctx context.Context, commentID int) (*domain.CommentReplyInfo, error) {
	if r.replyErr != nil {
		return nil, r.replyErr
	}
	if r.reply == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.reply, nil
}

func (r *fakeCommentRepository) CreateComment(ctx context.Context, comment *domain.VideoComment) error {
	if r.createErr != nil {
		return r.createErr
	}
	if comment == nil {
		return errors.New("comment is nil")
	}
	r.createCount++
	if comment.PCommentID == 0 {
		r.topLevelCreateCount++
	}
	copied := *comment
	copied.CommentID = 100
	r.created = &copied
	comment.CommentID = copied.CommentID
	return nil
}
