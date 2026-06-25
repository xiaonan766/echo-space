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
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
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
	if result.PCommentID != 7 || result.ReplyUserID != "30001" || result.ReplyNickName != "被回复用户" || result.ReplyAvatar != "reply-avatar.png" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestLoadCommentClosedCommentAreaReturnsNil(t *testing.T) {
	service := NewCommentService(&fakeCommentRepository{
		target: &domain.CommentTargetInfo{VideoID: "Abc123Def4", VideoUserID: "20001", Interaction: "1"},
	}, config.CommentReviewConfig{})

	result, err := service.LoadComment(context.Background(), LoadCommentInput{VideoID: "Abc123Def4"})
	if err != nil {
		t.Fatalf("load comment returned error: %v", err)
	}
	if result != nil {
		t.Fatalf("result = %#v, want nil", result)
	}
}

func TestLoadTopLevelCommentUsesCursorAndTopComment(t *testing.T) {
	repo := &fakeCommentRepository{
		target:        &domain.CommentTargetInfo{VideoID: "Abc123Def4", VideoUserID: "20001"},
		topLevelCount: 3,
		topComments: []domain.WebCommentItem{
			{CommentID: 9, TopType: 1},
		},
		topLevelComments: []domain.WebCommentItem{
			{CommentID: 8, LikeCount: 5},
			{CommentID: 7, LikeCount: 4},
			{CommentID: 6, LikeCount: 3},
		},
		replyCountMap: map[int]int{9: 2, 8: 1},
		actions: []domain.UserActionItem{
			{CommentID: 8, ActionType: 0, UserID: "10001"},
		},
	}
	service := NewCommentService(repo, config.CommentReviewConfig{})

	result, err := service.LoadComment(context.Background(), LoadCommentInput{
		VideoID:   "Abc123Def4",
		OrderType: commentOrderHot,
		UserID:    "10001",
	})
	if err != nil {
		t.Fatalf("load comment returned error: %v", err)
	}
	if len(result.CommentData.List) != 4 || result.CommentData.List[0].CommentID != 9 {
		t.Fatalf("list = %#v, want top comment first plus normal comments", result.CommentData.List)
	}
	if result.CommentData.List[0].ReplyCount != 2 || result.CommentData.List[1].ReplyCount != 1 {
		t.Fatalf("reply counts not filled: %#v", result.CommentData.List)
	}
	if len(result.UserActionList) != 1 || result.UserActionList[0].CommentID != 8 {
		t.Fatalf("user actions = %#v, want current page actions", result.UserActionList)
	}
}

func TestLoadCommentRejectsCursorMismatch(t *testing.T) {
	cursor, err := encodeCommentCursor(commentCursorPayload{
		VideoID:       "Other12345",
		PCommentID:    0,
		OrderType:     commentOrderHot,
		LastCommentID: 10,
		LastLikeCount: 5,
	})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	service := NewCommentService(&fakeCommentRepository{
		target: &domain.CommentTargetInfo{VideoID: "Abc123Def4", VideoUserID: "20001"},
	}, config.CommentReviewConfig{})
	_, err = service.LoadComment(context.Background(), LoadCommentInput{
		VideoID: "Abc123Def4",
		Cursor:  cursor,
	})
	if err == nil {
		t.Fatal("expected cursor mismatch error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %#v, want business error", err)
	}
}

type fakeCommentRepository struct {
	target *domain.CommentTargetInfo
	reply  *domain.CommentReplyInfo
	parent *domain.CommentReplyInfo

	targetErr error
	replyErr  error
	parentErr error
	createErr error

	topComments      []domain.WebCommentItem
	topLevelComments []domain.WebCommentItem
	replyComments    []domain.WebCommentItem
	topLevelCount    int64
	replyCount       int64
	replyCountMap    map[int]int
	actions          []domain.UserActionItem

	lastTopQuery   repository.CommentCursorQuery
	lastReplyQuery repository.CommentCursorQuery

	created             *domain.VideoComment
	createCount         int
	topLevelCreateCount int
}

func (r *fakeCommentRepository) FindCommentTarget(ctx context.Context, videoID string) (*domain.CommentTargetInfo, error) {
	if r.targetErr != nil {
		return nil, r.targetErr
	}
	if r.target == nil {
		return nil, gorm.ErrRecordNotFound
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

func (r *fakeCommentRepository) FindTopLevelComment(ctx context.Context, videoID string, commentID int) (*domain.CommentReplyInfo, error) {
	if r.parentErr != nil {
		return nil, r.parentErr
	}
	if r.parent == nil {
		return &domain.CommentReplyInfo{CommentID: commentID, VideoID: videoID}, nil
	}
	return r.parent, nil
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

func (r *fakeCommentRepository) ListTopComments(ctx context.Context, videoID string) ([]domain.WebCommentItem, error) {
	return r.topComments, nil
}

func (r *fakeCommentRepository) ListTopLevelCommentsByCursor(ctx context.Context, query repository.CommentCursorQuery) ([]domain.WebCommentItem, error) {
	r.lastTopQuery = query
	return r.topLevelComments, nil
}

func (r *fakeCommentRepository) ListReplyCommentsByCursor(ctx context.Context, query repository.CommentCursorQuery) ([]domain.WebCommentItem, error) {
	r.lastReplyQuery = query
	return r.replyComments, nil
}

func (r *fakeCommentRepository) CountTopLevelComments(ctx context.Context, videoID string) (int64, error) {
	return r.topLevelCount, nil
}

func (r *fakeCommentRepository) CountReplyComments(ctx context.Context, videoID string, pCommentID int) (int64, error) {
	return r.replyCount, nil
}

func (r *fakeCommentRepository) CountRepliesByParentIDs(ctx context.Context, parentIDs []int) (map[int]int, error) {
	if r.replyCountMap == nil {
		return map[int]int{}, nil
	}
	return r.replyCountMap, nil
}

func (r *fakeCommentRepository) ListUserCommentActions(ctx context.Context, videoID string, userID string, commentIDs []int) ([]domain.UserActionItem, error) {
	return r.actions, nil
}

func TestPostCommentUsesStableTimeFormat(t *testing.T) {
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
		Content:  "正常评论",
		VideoID:  "Abc123Def4",
	})
	if err != nil {
		t.Fatalf("post comment returned error: %v", err)
	}
	if result.PostTime != "2026-06-24 14:30:00" {
		t.Fatalf("postTime = %s, want formatted time", result.PostTime)
	}
}
