package web

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

func TestFocusUserRejectsSelfFocus(t *testing.T) {
	service := NewUhomeService(&fakeUhomeRepository{})

	err := service.FocusUser(context.Background(), "1000000001", "1000000001")
	if err == nil {
		t.Fatal("expected self focus error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %#v, want business error", err)
	}
}

func TestFocusUserReturnsSuccessWhenAlreadyFocused(t *testing.T) {
	repo := &fakeUhomeRepository{
		focus: &domain.UserFocus{UserID: "1000000001", FocusUserID: "1000000002"},
	}
	service := NewUhomeService(repo)

	if err := service.FocusUser(context.Background(), "1000000001", "1000000002"); err != nil {
		t.Fatalf("focus user returned error: %v", err)
	}
	if repo.createCount != 0 {
		t.Fatalf("createCount = %d, want 0", repo.createCount)
	}
}

func TestFocusUserRejectsMissingTargetUser(t *testing.T) {
	repo := &fakeUhomeRepository{
		focusErr: gorm.ErrRecordNotFound,
		userErr:  gorm.ErrRecordNotFound,
	}
	service := NewUhomeService(repo)

	err := service.FocusUser(context.Background(), "1000000001", "1000000002")
	if err == nil {
		t.Fatal("expected missing target user error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %#v, want business error", err)
	}
}

func TestFocusUserCreatesFocus(t *testing.T) {
	repo := &fakeUhomeRepository{
		focusErr: gorm.ErrRecordNotFound,
		user:     &domain.UserInfo{UserID: "1000000002"},
	}
	service := NewUhomeService(repo)
	service.now = func() time.Time {
		return time.Date(2026, 6, 25, 10, 30, 0, 0, time.Local)
	}

	if err := service.FocusUser(context.Background(), "1000000001", "1000000002"); err != nil {
		t.Fatalf("focus user returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected created focus")
	}
	if repo.created.UserID != "1000000001" || repo.created.FocusUserID != "1000000002" {
		t.Fatalf("created focus = %#v, want user/focus pair", repo.created)
	}
	if !repo.created.FocusTime.Equal(service.now()) {
		t.Fatalf("focusTime = %s, want fixed time", repo.created.FocusTime)
	}
}

func TestGetUserInfoReturnsAnonymousUserHomeInfo(t *testing.T) {
	repo := &fakeUhomeRepository{
		user: &domain.UserInfo{
			UserID:      "1000000002",
			NickName:    "UP主",
			Sex:         1,
			PersonIntro: "简介",
			NoticeInfo:  "公告",
			Theme:       2,
			Avatar:      "avatar.png",
		},
		videoCountInfo: domain.UserVideoCountInfo{PlayCount: 30, LikeCount: 7},
		fansCount:      12,
		focusCount:     5,
		focusErr:       gorm.ErrRecordNotFound,
	}
	service := NewUhomeService(repo)

	result, err := service.GetUserInfo(context.Background(), "", "1000000002")
	if err != nil {
		t.Fatalf("get user info returned error: %v", err)
	}
	if result.UserID != "1000000002" || result.NickName != "UP主" {
		t.Fatalf("result = %#v, want target user info", result)
	}
	if result.PlayCount != 30 || result.LikeCount != 7 || result.FansCount != 12 || result.FocusCount != 5 {
		t.Fatalf("count info = %#v, want aggregated counts", result)
	}
	if result.HaveFocus {
		t.Fatal("haveFocus = true, want false for anonymous user")
	}
}

func TestGetUserInfoMarksFocusedUser(t *testing.T) {
	repo := &fakeUhomeRepository{
		user:  &domain.UserInfo{UserID: "1000000002", NickName: "UP主"},
		focus: &domain.UserFocus{UserID: "1000000001", FocusUserID: "1000000002"},
	}
	service := NewUhomeService(repo)

	result, err := service.GetUserInfo(context.Background(), "1000000001", "1000000002")
	if err != nil {
		t.Fatalf("get user info returned error: %v", err)
	}
	if !result.HaveFocus {
		t.Fatal("haveFocus = false, want true")
	}
}

func TestGetUserInfoReturnsNotFoundWhenUserMissing(t *testing.T) {
	repo := &fakeUhomeRepository{
		userErr: gorm.ErrRecordNotFound,
	}
	service := NewUhomeService(repo)

	_, err := service.GetUserInfo(context.Background(), "", "1000000002")
	if err == nil {
		t.Fatal("expected missing user error")
	}
	if _, ok := IsNotFoundError(err); !ok {
		t.Fatalf("error = %#v, want not found error", err)
	}
}

func TestUpdateUserInfoRejectsOtherUser(t *testing.T) {
	repo := &fakeUhomeRepository{}
	service := NewUhomeService(repo)

	_, err := service.UpdateUserInfo(context.Background(), "1000000001", UpdateUserInfoInput{
		UserID:   "1000000002",
		Avatar:   "avatar.png",
		NickName: "newName",
		Sex:      1,
	})
	if err == nil {
		t.Fatal("expected other user error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %#v, want business error", err)
	}
	if repo.updateCount != 0 {
		t.Fatalf("updateCount = %d, want 0", repo.updateCount)
	}
}

func TestUpdateUserInfoRejectsInsufficientCoinForNickName(t *testing.T) {
	repo := &fakeUhomeRepository{
		user: &domain.UserInfo{
			UserID:           "1000000001",
			NickName:         "oldName",
			CurrentCoinCount: 4,
		},
	}
	service := NewUhomeService(repo)

	_, err := service.UpdateUserInfo(context.Background(), "1000000001", UpdateUserInfoInput{
		UserID:   "1000000001",
		Avatar:   "avatar.png",
		NickName: "newName",
		Sex:      1,
	})
	if err == nil {
		t.Fatal("expected insufficient coin error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %#v, want business error", err)
	}
	if repo.updateCount != 0 {
		t.Fatalf("updateCount = %d, want 0", repo.updateCount)
	}
}

func TestUpdateUserInfoRejectsDuplicateNickName(t *testing.T) {
	repo := &fakeUhomeRepository{
		user: &domain.UserInfo{
			UserID:           "1000000001",
			NickName:         "oldName",
			CurrentCoinCount: 10,
		},
		nickNameUser: &domain.UserInfo{
			UserID:   "1000000002",
			NickName: "newName",
		},
	}
	service := NewUhomeService(repo)

	_, err := service.UpdateUserInfo(context.Background(), "1000000001", UpdateUserInfoInput{
		UserID:   "1000000001",
		Avatar:   "avatar.png",
		NickName: "newName",
		Sex:      1,
	})
	if err == nil {
		t.Fatal("expected duplicate nick name error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %#v, want business error", err)
	}
	if repo.updateCount != 0 {
		t.Fatalf("updateCount = %d, want 0", repo.updateCount)
	}
}

func TestUpdateUserInfoChargesCoinWhenNickNameChanges(t *testing.T) {
	repo := &fakeUhomeRepository{
		user: &domain.UserInfo{
			UserID:           "1000000001",
			NickName:         "oldName",
			CurrentCoinCount: 10,
			Theme:            1,
		},
		nickNameErr: gorm.ErrRecordNotFound,
	}
	service := NewUhomeService(repo)

	result, err := service.UpdateUserInfo(context.Background(), "1000000001", UpdateUserInfoInput{
		UserID:             "1000000001",
		Avatar:             "avatar.png",
		NickName:           "newName",
		Sex:                2,
		Birthday:           "2026-06-25",
		School:             "school",
		PersonIntroduction: "intro",
		NoticeInfo:         "notice",
	})
	if err != nil {
		t.Fatalf("update user info returned error: %v", err)
	}
	if repo.updateCount != 1 {
		t.Fatalf("updateCount = %d, want 1", repo.updateCount)
	}
	if repo.updateSpendCoin != updateNickNameCoin {
		t.Fatalf("updateSpendCoin = %d, want %d", repo.updateSpendCoin, updateNickNameCoin)
	}
	if result.NickName != "newName" || result.CurrentCoinCount != 5 {
		t.Fatalf("result = %#v, want changed nick name and charged coin", result)
	}
}

func TestUpdateUserInfoDoesNotChargeWhenNickNameUnchanged(t *testing.T) {
	repo := &fakeUhomeRepository{
		user: &domain.UserInfo{
			UserID:           "1000000001",
			NickName:         "oldName",
			CurrentCoinCount: 10,
		},
	}
	service := NewUhomeService(repo)

	_, err := service.UpdateUserInfo(context.Background(), "1000000001", UpdateUserInfoInput{
		UserID:   "1000000001",
		Avatar:   "avatar.png",
		NickName: "oldName",
		Sex:      1,
	})
	if err != nil {
		t.Fatalf("update user info returned error: %v", err)
	}
	if repo.updateSpendCoin != 0 {
		t.Fatalf("updateSpendCoin = %d, want 0", repo.updateSpendCoin)
	}
}

type fakeUhomeRepository struct {
	user  *domain.UserInfo
	focus *domain.UserFocus

	userErr     error
	nickNameErr error
	focusErr    error
	createErr   error

	nickNameUser *domain.UserInfo

	videoCountInfo domain.UserVideoCountInfo
	fansCount      int64
	focusCount     int64
	videoCountErr  error
	fansCountErr   error
	focusCountErr  error

	created     *domain.UserFocus
	createCount int

	updatedUser     *domain.UserInfo
	updateInput     domain.UserInfo
	updateUserID    string
	updateSpendCoin int
	updateErr       error
	updateCount     int
}

func (r *fakeUhomeRepository) FindByUserID(ctx context.Context, userID string) (*domain.UserInfo, error) {
	if r.userErr != nil {
		return nil, r.userErr
	}
	if r.user == nil {
		return nil, errors.New("user not configured")
	}
	return r.user, nil
}

func (r *fakeUhomeRepository) FindByNickName(ctx context.Context, nickName string) (*domain.UserInfo, error) {
	if r.nickNameErr != nil {
		return nil, r.nickNameErr
	}
	if r.nickNameUser == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.nickNameUser, nil
}

func (r *fakeUhomeRepository) UpdateUserInfo(ctx context.Context, userID string, userInfo domain.UserInfo, spendCoin int) (*domain.UserInfo, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}

	r.updateCount++
	r.updateUserID = userID
	r.updateInput = userInfo
	r.updateSpendCoin = spendCoin
	if r.updatedUser != nil {
		return r.updatedUser, nil
	}

	updatedUser := domain.UserInfo{}
	if r.user != nil {
		updatedUser = *r.user
	}
	updatedUser.UserID = userID
	updatedUser.Avatar = userInfo.Avatar
	updatedUser.NickName = userInfo.NickName
	updatedUser.Sex = userInfo.Sex
	updatedUser.Birthday = userInfo.Birthday
	updatedUser.School = userInfo.School
	updatedUser.PersonIntro = userInfo.PersonIntro
	updatedUser.NoticeInfo = userInfo.NoticeInfo
	updatedUser.CurrentCoinCount -= spendCoin
	return &updatedUser, nil
}

func (r *fakeUhomeRepository) FindFocus(ctx context.Context, userID string, focusUserID string) (*domain.UserFocus, error) {
	if r.focusErr != nil {
		return nil, r.focusErr
	}
	if r.focus == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.focus, nil
}

func (r *fakeUhomeRepository) CreateFocus(ctx context.Context, focus *domain.UserFocus) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.createCount++
	copied := *focus
	r.created = &copied
	return nil
}

func (r *fakeUhomeRepository) CountFocus(ctx context.Context, userID string) (int64, error) {
	if r.focusCountErr != nil {
		return 0, r.focusCountErr
	}
	return r.focusCount, nil
}

func (r *fakeUhomeRepository) CountFans(ctx context.Context, userID string) (int64, error) {
	if r.fansCountErr != nil {
		return 0, r.fansCountErr
	}
	return r.fansCount, nil
}

func (r *fakeUhomeRepository) SumUserVideoCount(ctx context.Context, userID string) (domain.UserVideoCountInfo, error) {
	if r.videoCountErr != nil {
		return domain.UserVideoCountInfo{}, r.videoCountErr
	}
	return r.videoCountInfo, nil
}
