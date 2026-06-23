package web

import (
	"context"
	"testing"
)

func TestLoadVideoPListValidatesVideoID(t *testing.T) {
	service := NewVideoService(nil)
	_, err := service.LoadVideoPList(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %T %v, want BusinessError", err, err)
	}
}
