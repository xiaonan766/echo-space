package web

import (
	"errors"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

func TestValidateDoUserActionInput(t *testing.T) {
	valid := DoUserActionInput{
		UserID:      "10001",
		VideoID:     "Abc123Def4",
		ActionType:  actionVideoLike,
		ActionCount: 1,
	}
	if err := validateDoUserActionInput(valid); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*DoUserActionInput)
	}{
		{name: "empty user", mutate: func(input *DoUserActionInput) { input.UserID = "" }},
		{name: "invalid video id", mutate: func(input *DoUserActionInput) { input.VideoID = "bad" }},
		{name: "unsupported action type", mutate: func(input *DoUserActionInput) { input.ActionType = 7 }},
		{name: "invalid action count", mutate: func(input *DoUserActionInput) { input.ActionCount = 3 }},
		{name: "comment action without comment id", mutate: func(input *DoUserActionInput) { input.ActionType = actionCommentLike }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if err := validateDoUserActionInput(input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeDoUserActionInput(t *testing.T) {
	input := normalizeDoUserActionInput(DoUserActionInput{
		UserID:     " 10001 ",
		VideoID:    " Abc123Def4 ",
		ActionType: actionVideoCollect,
		CommentID:  99,
	})
	if input.UserID != "10001" || input.VideoID != "Abc123Def4" {
		t.Fatalf("input was not trimmed: %#v", input)
	}
	if input.ActionCount != 1 {
		t.Fatalf("action count = %d, want 1", input.ActionCount)
	}
	if input.CommentID != 0 {
		t.Fatalf("video action comment id = %d, want 0", input.CommentID)
	}
}

func TestMapUserActionRepositoryError(t *testing.T) {
	err := mapUserActionRepositoryError(repository.ErrUserActionInsufficientCoin)
	businessError, ok := IsBusinessError(err)
	if !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
	if businessError.Info != "\u786c\u5e01\u4e0d\u8db3" {
		t.Fatalf("business info = %q", businessError.Info)
	}

	rawErr := errors.New("raw")
	if got := mapUserActionRepositoryError(rawErr); !errors.Is(got, rawErr) {
		t.Fatalf("raw error was not preserved")
	}
}
