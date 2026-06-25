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

type fakeUhomeRepository struct {
	user  *domain.UserInfo
	focus *domain.UserFocus

	userErr   error
	focusErr  error
	createErr error

	created     *domain.UserFocus
	createCount int
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
