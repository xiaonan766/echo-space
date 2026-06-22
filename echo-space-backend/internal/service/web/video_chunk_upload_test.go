package web

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

const testUploadID = "Abc123Def456Ghi"

type fakeVideoUploadSettingStore struct {
	setting domain.SysSetting
	exists  bool
	err     error
}

func (s *fakeVideoUploadSettingStore) Get(_ context.Context) (domain.SysSetting, bool, error) {
	return s.setting, s.exists, s.err
}

func TestVideoUploadServiceUploadsSequentialChunks(t *testing.T) {
	service, store, uploadDir := newChunkUploadTestService(t, 3, 1)
	uploadTestChunk(t, service, 0, []byte("first"))
	uploadTestChunk(t, service, 1, []byte("second"))

	if store.info.ChunkIndex != 1 || store.info.FileSize != 11 {
		t.Fatalf("metadata = %+v", store.info)
	}
	assertFileContent(t, filepath.Join(uploadDir, "0"), []byte("first"))
	assertFileContent(t, filepath.Join(uploadDir, "1"), []byte("second"))
}

func TestVideoUploadServiceReuploadsChunkIdempotently(t *testing.T) {
	service, store, uploadDir := newChunkUploadTestService(t, 2, 1)
	uploadTestChunk(t, service, 0, []byte("old"))
	uploadTestChunk(t, service, 0, []byte("replacement"))

	if store.info.ChunkIndex != 0 || store.info.FileSize != int64(len("replacement")) {
		t.Fatalf("metadata = %+v", store.info)
	}
	assertFileContent(t, filepath.Join(uploadDir, "0"), []byte("replacement"))
}

func TestVideoUploadServiceRejectsInvalidChunkOrderAndRange(t *testing.T) {
	service, _, _ := newChunkUploadTestService(t, 3, 1)
	tests := []struct {
		name       string
		chunkIndex int
	}{
		{name: "negative", chunkIndex: -1},
		{name: "skip first chunk", chunkIndex: 1},
		{name: "out of range", chunkIndex: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fileHeader, cleanup := newChunkFileHeader(t, []byte("chunk"))
			defer cleanup()
			err := service.UploadVideoChunk(context.Background(), UploadVideoChunkInput{
				UserID: "user", UploadID: testUploadID, ChunkIndex: test.chunkIndex, ChunkFile: fileHeader,
			})
			if _, ok := IsBusinessError(err); !ok {
				t.Fatalf("error = %v, want BusinessError", err)
			}
		})
	}
}

func TestVideoUploadServiceRejectsExpiredUpload(t *testing.T) {
	service, store, _ := newChunkUploadTestService(t, 2, 1)
	store.exists = false
	fileHeader, cleanup := newChunkFileHeader(t, []byte("chunk"))
	defer cleanup()

	err := service.UploadVideoChunk(context.Background(), UploadVideoChunkInput{
		UserID: "user", UploadID: testUploadID, ChunkIndex: 0, ChunkFile: fileHeader,
	})
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %v, want BusinessError", err)
	}
}

func TestVideoUploadServiceRejectsVideoOverSizeLimit(t *testing.T) {
	service, store, uploadDir := newChunkUploadTestService(t, 2, 1)
	firstChunk := bytes.Repeat([]byte{'a'}, bytesPerMB-1)
	uploadTestChunk(t, service, 0, firstChunk)
	updateCalls := store.updateCalls

	fileHeader, cleanup := newChunkFileHeader(t, []byte("too-large"))
	defer cleanup()
	err := service.UploadVideoChunk(context.Background(), UploadVideoChunkInput{
		UserID: "user", UploadID: testUploadID, ChunkIndex: 1, ChunkFile: fileHeader,
	})
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %v, want BusinessError", err)
	}
	if store.updateCalls != updateCalls || store.info.FileSize != int64(len(firstChunk)) {
		t.Fatalf("metadata changed after rejected chunk: %+v", store.info)
	}
	if _, statErr := os.Stat(filepath.Join(uploadDir, "1")); !os.IsNotExist(statErr) {
		t.Fatalf("rejected chunk exists: %v", statErr)
	}
}

func TestVideoUploadServiceRecoversMetadataAfterRedisError(t *testing.T) {
	service, store, uploadDir := newChunkUploadTestService(t, 2, 1)
	store.updateErr = errors.New("redis unavailable")
	fileHeader, cleanup := newChunkFileHeader(t, []byte("chunk"))
	err := service.UploadVideoChunk(context.Background(), UploadVideoChunkInput{
		UserID: "user", UploadID: testUploadID, ChunkIndex: 0, ChunkFile: fileHeader,
	})
	cleanup()
	if err == nil {
		t.Fatal("UploadVideoChunk() error = nil")
	}
	assertFileContent(t, filepath.Join(uploadDir, "0"), []byte("chunk"))

	store.updateErr = nil
	uploadTestChunk(t, service, 0, []byte("chunk"))
	if store.info.ChunkIndex != 0 || store.info.FileSize != 5 {
		t.Fatalf("metadata not recovered: %+v", store.info)
	}
}

func TestVideoUploadServiceRollsBackWhenMetadataExpiresDuringUpload(t *testing.T) {
	service, store, uploadDir := newChunkUploadTestService(t, 2, 1)
	store.updateExists = false
	fileHeader, cleanup := newChunkFileHeader(t, []byte("chunk"))
	defer cleanup()

	err := service.UploadVideoChunk(context.Background(), UploadVideoChunkInput{
		UserID: "user", UploadID: testUploadID, ChunkIndex: 0, ChunkFile: fileHeader,
	})
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %v, want BusinessError", err)
	}
	if _, statErr := os.Stat(filepath.Join(uploadDir, "0")); !os.IsNotExist(statErr) {
		t.Fatalf("chunk was not rolled back: %v", statErr)
	}
}

func TestVideoUploadServiceSerializesSameChunk(t *testing.T) {
	service, store, uploadDir := newChunkUploadTestService(t, 2, 1)
	files := make([]*multipart.FileHeader, 2)
	cleanups := make([]func(), 2)
	for index := range files {
		files[index], cleanups[index] = newChunkFileHeader(t, []byte("chunk"))
		defer cleanups[index]()
	}

	var waitGroup sync.WaitGroup
	errorsCh := make(chan error, len(files))
	for _, fileHeader := range files {
		waitGroup.Add(1)
		go func(header *multipart.FileHeader) {
			defer waitGroup.Done()
			errorsCh <- service.UploadVideoChunk(context.Background(), UploadVideoChunkInput{
				UserID: "user", UploadID: testUploadID, ChunkIndex: 0, ChunkFile: header,
			})
		}(fileHeader)
	}
	waitGroup.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent upload error = %v", err)
		}
	}
	if store.info.FileSize != 5 || store.info.ChunkIndex != 0 {
		t.Fatalf("metadata = %+v", store.info)
	}
	assertFileContent(t, filepath.Join(uploadDir, "0"), []byte("chunk"))
}

func newChunkUploadTestService(t *testing.T, chunks int, videoSizeMB int) (*VideoUploadService, *fakeUploadingFileStore, string) {
	t.Helper()
	resourceRoot := t.TempDir()
	filePath := "20260622/user" + testUploadID
	uploadDir := filepath.Join(resourceRoot, "temp", filepath.FromSlash(filePath))
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		t.Fatalf("create upload directory: %v", err)
	}
	store := &fakeUploadingFileStore{
		info: cache.UploadingFileInfo{
			Chunks: chunks, FileName: "video", UploadID: testUploadID, ChunkIndex: 0, FilePath: filePath,
		},
		exists:       true,
		updateExists: true,
	}
	settingStore := &fakeVideoUploadSettingStore{
		setting: domain.SysSetting{VideoSize: videoSizeMB},
		exists:  true,
	}
	return NewVideoUploadService(store, settingStore, resourceRoot), store, uploadDir
}

func uploadTestChunk(t *testing.T, service *VideoUploadService, chunkIndex int, content []byte) {
	t.Helper()
	fileHeader, cleanup := newChunkFileHeader(t, content)
	defer cleanup()
	if err := service.UploadVideoChunk(context.Background(), UploadVideoChunkInput{
		UserID: "user", UploadID: testUploadID, ChunkIndex: chunkIndex, ChunkFile: fileHeader,
	}); err != nil {
		t.Fatalf("UploadVideoChunk(%d) error = %v", chunkIndex, err)
	}
}

func newChunkFileHeader(t *testing.T, content []byte) (*multipart.FileHeader, func()) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("chunkFile", "chunk.bin")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest("POST", "/file/uploadVideo", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(int64(len(content)) + 1024); err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}
	fileHeader := request.MultipartForm.File["chunkFile"][0]
	cleanup := func() { _ = request.MultipartForm.RemoveAll() }
	return fileHeader, cleanup
}

func assertFileContent(t *testing.T, path string, expected []byte) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Equal(content, expected) {
		t.Fatalf("content = %q, want %q", content, expected)
	}
}
