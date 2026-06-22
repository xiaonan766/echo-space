package web

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

type uploadingFileStoreCall struct {
	userID   string
	uploadID string
	info     cache.UploadingFileInfo
	ttl      time.Duration
}

type fakeUploadingFileStore struct {
	results      []bool
	err          error
	calls        []uploadingFileStoreCall
	info         cache.UploadingFileInfo
	exists       bool
	getErr       error
	updateErr    error
	updateExists bool
	updateCalls  int
}

func (s *fakeUploadingFileStore) Create(_ context.Context, userID string, uploadID string, info cache.UploadingFileInfo, ttl time.Duration) (bool, error) {
	s.calls = append(s.calls, uploadingFileStoreCall{
		userID:   userID,
		uploadID: uploadID,
		info:     info,
		ttl:      ttl,
	})
	if s.err != nil {
		return false, s.err
	}
	index := len(s.calls) - 1
	if index < len(s.results) {
		return s.results[index], nil
	}
	return true, nil
}

func (s *fakeUploadingFileStore) Get(_ context.Context, _ string, _ string) (cache.UploadingFileInfo, bool, error) {
	if s.getErr != nil {
		return cache.UploadingFileInfo{}, false, s.getErr
	}
	return s.info, s.exists, nil
}

func (s *fakeUploadingFileStore) UpdateIfExists(_ context.Context, _ string, _ string, info cache.UploadingFileInfo, _ time.Duration) (bool, error) {
	s.updateCalls++
	if s.updateErr != nil {
		return false, s.updateErr
	}
	if !s.updateExists {
		return false, nil
	}
	s.info = info
	return true, nil
}

func TestVideoUploadServicePreUploadVideo(t *testing.T) {
	resourceRoot := t.TempDir()
	store := &fakeUploadingFileStore{}
	service := newTestVideoUploadService(resourceRoot, store, "Abc123Def456Ghi")

	uploadID, err := service.PreUploadVideo(context.Background(), PreUploadVideoInput{
		UserID:   "5171840855",
		FileName: "demo-video",
		Chunks:   8,
	})
	if err != nil {
		t.Fatalf("PreUploadVideo() error = %v", err)
	}
	if uploadID != "Abc123Def456Ghi" {
		t.Fatalf("uploadID = %q", uploadID)
	}
	if len(store.calls) != 1 {
		t.Fatalf("store calls = %d", len(store.calls))
	}

	call := store.calls[0]
	if call.userID != "5171840855" || call.uploadID != uploadID {
		t.Fatalf("unexpected store identity: %+v", call)
	}
	if call.ttl != 24*time.Hour {
		t.Fatalf("ttl = %v", call.ttl)
	}
	if call.info.Chunks != 8 || call.info.FileName != "demo-video" || call.info.ChunkIndex != 0 {
		t.Fatalf("unexpected metadata: %+v", call.info)
	}
	if call.info.FilePath != "20260622/5171840855Abc123Def456Ghi" {
		t.Fatalf("filePath = %q", call.info.FilePath)
	}

	uploadDir := filepath.Join(resourceRoot, "temp", "20260622", "5171840855Abc123Def456Ghi")
	if stat, err := os.Stat(uploadDir); err != nil || !stat.IsDir() {
		t.Fatalf("upload directory not created: stat=%v err=%v", stat, err)
	}
}

func TestVideoUploadServiceValidatesInput(t *testing.T) {
	tests := []struct {
		name  string
		input PreUploadVideoInput
	}{
		{name: "missing user", input: PreUploadVideoInput{FileName: "video", Chunks: 1}},
		{name: "missing file name", input: PreUploadVideoInput{UserID: "user", Chunks: 1}},
		{name: "file name too long", input: PreUploadVideoInput{UserID: "user", FileName: strings.Repeat("名", 256), Chunks: 1}},
		{name: "invalid chunks", input: PreUploadVideoInput{UserID: "user", FileName: "video", Chunks: 0}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeUploadingFileStore{}
			service := newTestVideoUploadService(t.TempDir(), store, "Abc123Def456Ghi")
			_, err := service.PreUploadVideo(context.Background(), test.input)
			if _, ok := IsBusinessError(err); !ok {
				t.Fatalf("error = %v, want BusinessError", err)
			}
			if len(store.calls) != 0 {
				t.Fatalf("store calls = %d", len(store.calls))
			}
		})
	}
}

func TestVideoUploadServiceRetriesRedisCollision(t *testing.T) {
	resourceRoot := t.TempDir()
	store := &fakeUploadingFileStore{results: []bool{false, true}}
	service := NewVideoUploadService(store, nil, resourceRoot)
	service.now = func() time.Time { return time.Date(2026, 6, 22, 15, 0, 0, 0, time.Local) }
	ids := []string{"First123456789A", "Second12345678B"}
	service.generateID = func(_ int) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}

	uploadID, err := service.PreUploadVideo(context.Background(), PreUploadVideoInput{
		UserID: "user", FileName: "video", Chunks: 2,
	})
	if err != nil {
		t.Fatalf("PreUploadVideo() error = %v", err)
	}
	if uploadID != "Second12345678B" {
		t.Fatalf("uploadID = %q", uploadID)
	}
	if len(store.calls) != 2 {
		t.Fatalf("store calls = %d", len(store.calls))
	}
	firstDir := filepath.Join(resourceRoot, "temp", "20260622", "userFirst123456789A")
	if _, err := os.Stat(firstDir); !os.IsNotExist(err) {
		t.Fatalf("collided directory still exists: %v", err)
	}
}

func TestVideoUploadServiceCleansDirectoryWhenRedisFails(t *testing.T) {
	resourceRoot := t.TempDir()
	store := &fakeUploadingFileStore{err: errors.New("redis unavailable")}
	service := newTestVideoUploadService(resourceRoot, store, "Abc123Def456Ghi")

	_, err := service.PreUploadVideo(context.Background(), PreUploadVideoInput{
		UserID: "user", FileName: "video", Chunks: 2,
	})
	if err == nil {
		t.Fatal("PreUploadVideo() error = nil")
	}
	uploadDir := filepath.Join(resourceRoot, "temp", "20260622", "userAbc123Def456Ghi")
	if _, statErr := os.Stat(uploadDir); !os.IsNotExist(statErr) {
		t.Fatalf("upload directory still exists: %v", statErr)
	}
}

func TestVideoUploadServiceDoesNotWriteRedisWhenDirectoryCreationFails(t *testing.T) {
	resourceRoot := filepath.Join(t.TempDir(), "resource-file")
	if err := os.WriteFile(resourceRoot, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("write resource file: %v", err)
	}
	store := &fakeUploadingFileStore{}
	service := newTestVideoUploadService(resourceRoot, store, "Abc123Def456Ghi")

	_, err := service.PreUploadVideo(context.Background(), PreUploadVideoInput{
		UserID: "user", FileName: "video", Chunks: 2,
	})
	if err == nil {
		t.Fatal("PreUploadVideo() error = nil")
	}
	if len(store.calls) != 0 {
		t.Fatalf("store calls = %d", len(store.calls))
	}
}

func newTestVideoUploadService(resourceRoot string, store UploadingFileMetadataStore, uploadID string) *VideoUploadService {
	service := NewVideoUploadService(store, nil, resourceRoot)
	service.now = func() time.Time { return time.Date(2026, 6, 22, 15, 0, 0, 0, time.Local) }
	service.generateID = func(_ int) (string, error) { return uploadID, nil }
	return service
}
