package admin

import (
	"context"
	"strings"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

func TestAuditVideoValidatesInput(t *testing.T) {
	tests := []struct {
		name  string
		input AuditVideoInput
	}{
		{
			name: "invalid status",
			input: AuditVideoInput{
				VideoID: "Abc123Def4",
				Status:  domain.VideoPostStatusPendingReview,
			},
		},
		{
			name: "missing reject reason",
			input: AuditVideoInput{
				VideoID: "Abc123Def4",
				Status:  domain.VideoPostStatusRejected,
			},
		},
		{
			name: "long reject reason",
			input: AuditVideoInput{
				VideoID: "Abc123Def4",
				Status:  domain.VideoPostStatusRejected,
				Reason:  strings.Repeat("拒", 201),
			},
		},
		{
			name: "invalid video id",
			input: AuditVideoInput{
				VideoID: "bad-id",
				Status:  domain.VideoPostStatusApproved,
			},
		},
	}

	service := NewVideoInfoService(nil)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := service.AuditVideo(context.Background(), test.input); err == nil {
				t.Fatal("expected business error")
			} else if _, ok := IsBusinessError(err); !ok {
				t.Fatalf("error = %T %v, want BusinessError", err, err)
			}
		})
	}
}

func TestLoadVideoPListValidatesVideoID(t *testing.T) {
	service := NewVideoInfoService(nil)
	_, err := service.LoadVideoPList(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %T %v, want BusinessError", err, err)
	}
}

func TestRecommendVideoValidatesVideoID(t *testing.T) {
	service := NewVideoInfoService(nil)
	err := service.RecommendVideo(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %T %v, want BusinessError", err, err)
	}
}
