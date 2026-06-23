package web

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math/big"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

const (
	uploadIDLength          = 15
	maxUploadIDAttempts     = 3
	maxUploadFileNameLength = 255
	uploadingFileTTL        = 24 * time.Hour
	defaultResourceRoot     = "resources"
	temporaryVideoDir       = "temp"
	uploadLockStripeCount   = 256
	bytesPerMB              = 1024 * 1024
)

const uploadIDAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type UploadingFileMetadataStore interface {
	Create(ctx context.Context, userID string, uploadID string, info cache.UploadingFileInfo, ttl time.Duration) (bool, error)
	Get(ctx context.Context, userID string, uploadID string) (cache.UploadingFileInfo, bool, error)
	UpdateIfExists(ctx context.Context, userID string, uploadID string, info cache.UploadingFileInfo, ttl time.Duration) (bool, error)
}

type VideoUploadSettingStore interface {
	Get(ctx context.Context) (domain.SysSetting, bool, error)
}

type VideoUploadService struct {
	store        UploadingFileMetadataStore
	settingStore VideoUploadSettingStore
	tempRoot     string
	now          func() time.Time
	generateID   func(int) (string, error)
	uploadLocks  [uploadLockStripeCount]sync.Mutex
}

type PreUploadVideoInput struct {
	UserID   string
	FileName string
	Chunks   int
}

type UploadVideoChunkInput struct {
	UserID     string
	UploadID   string
	ChunkIndex int
	ChunkFile  *multipart.FileHeader
}

type uploadDiskState struct {
	TotalSize         int64
	HighestContiguous int
	ChunkSizes        map[int]int64
}

func NewVideoUploadService(store UploadingFileMetadataStore, settingStore VideoUploadSettingStore, resourceRoot string) *VideoUploadService {
	resourceRoot = strings.TrimSpace(resourceRoot)
	if resourceRoot == "" {
		resourceRoot = defaultResourceRoot
	}
	return &VideoUploadService{
		store:        store,
		settingStore: settingStore,
		tempRoot:     filepath.Join(resourceRoot, temporaryVideoDir),
		now:          time.Now,
		generateID:   randomUploadID,
	}
}

func (s *VideoUploadService) PreUploadVideo(ctx context.Context, input PreUploadVideoInput) (string, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.FileName = strings.TrimSpace(input.FileName)
	if input.UserID == "" {
		return "", &BusinessError{Info: "请先登录"}
	}
	if input.FileName == "" {
		return "", &BusinessError{Info: "请输入文件名称"}
	}
	if utf8.RuneCountInString(input.FileName) > maxUploadFileNameLength {
		return "", &BusinessError{Info: "文件名称不能超过255个字符"}
	}
	if input.Chunks <= 0 {
		return "", &BusinessError{Info: "文件分片数量必须大于0"}
	}
	if s == nil || s.store == nil {
		return "", fmt.Errorf("video upload service is not ready")
	}

	day := s.now().Format("20060102")
	for attempt := 0; attempt < maxUploadIDAttempts; attempt++ {
		uploadID, err := s.generateID(uploadIDLength)
		if err != nil {
			return "", fmt.Errorf("generate upload id: %w", err)
		}

		filePath := filepath.ToSlash(filepath.Join(day, input.UserID+uploadID))
		uploadDir := filepath.Join(s.tempRoot, filepath.FromSlash(filePath))
		created, err := createUploadDirectory(uploadDir)
		if err != nil {
			return "", fmt.Errorf("create upload directory: %w", err)
		}
		if !created {
			continue
		}

		info := cache.UploadingFileInfo{
			Chunks:     input.Chunks,
			FileName:   input.FileName,
			UploadID:   uploadID,
			ChunkIndex: 0,
			FileSize:   0,
			FilePath:   filePath,
		}
		stored, storeErr := s.store.Create(ctx, input.UserID, uploadID, info, uploadingFileTTL)
		if storeErr != nil {
			return "", cleanupUploadDirectory(uploadDir, fmt.Errorf("save uploading file metadata: %w", storeErr))
		}
		if !stored {
			if err := os.RemoveAll(uploadDir); err != nil {
				return "", fmt.Errorf("cleanup collided upload directory: %w", err)
			}
			continue
		}
		return uploadID, nil
	}

	return "", fmt.Errorf("generate unique upload id failed after %d attempts", maxUploadIDAttempts)
}

func (s *VideoUploadService) UploadVideoChunk(ctx context.Context, input UploadVideoChunkInput) error {
	input.UserID = strings.TrimSpace(input.UserID)
	input.UploadID = strings.TrimSpace(input.UploadID)
	if input.UserID == "" {
		return &BusinessError{Info: "请先登录"}
	}
	if !validUploadID(input.UploadID) {
		return &BusinessError{Info: "上传任务不存在，请重新上传"}
	}
	if input.ChunkIndex < 0 {
		return &BusinessError{Info: "分片索引不正确"}
	}
	if input.ChunkFile == nil {
		return &BusinessError{Info: "请选择要上传的视频分片"}
	}
	if s == nil || s.store == nil {
		return fmt.Errorf("video upload service is not ready")
	}

	lock := s.uploadLock(input.UserID, input.UploadID)
	lock.Lock()
	defer lock.Unlock()

	info, exists, err := s.store.Get(ctx, input.UserID, input.UploadID)
	if err != nil {
		return fmt.Errorf("get uploading file metadata: %w", err)
	}
	if !exists {
		return &BusinessError{Info: "文件不存在，请重新上传"}
	}
	if info.UploadID != input.UploadID || info.Chunks <= 0 {
		return &BusinessError{Info: "上传任务信息异常，请重新上传"}
	}
	if input.ChunkIndex >= info.Chunks {
		return &BusinessError{Info: "分片索引超出范围"}
	}

	uploadDir, err := s.safeUploadDirectory(info.FilePath)
	if err != nil {
		return fmt.Errorf("resolve upload directory: %w", err)
	}
	state, err := scanUploadDiskState(uploadDir, info.Chunks)
	if errors.Is(err, os.ErrNotExist) {
		return &BusinessError{Info: "文件不存在，请重新上传"}
	}
	if err != nil {
		return fmt.Errorf("scan upload chunks: %w", err)
	}
	if input.ChunkIndex > state.HighestContiguous+1 {
		return &BusinessError{Info: "请按顺序上传视频分片"}
	}

	maxVideoSize, err := s.maxVideoSize(ctx)
	if err != nil {
		return fmt.Errorf("get video size setting: %w", err)
	}
	oldChunkSize := state.ChunkSizes[input.ChunkIndex]
	baseSize := state.TotalSize - oldChunkSize
	remainingSize := maxVideoSize - baseSize
	if remainingSize < 0 {
		return &BusinessError{Info: "文件超过大小限制"}
	}

	tempPath, written, err := writeChunkToTemp(input.ChunkFile, uploadDir, input.ChunkIndex, remainingSize)
	if errors.Is(err, errVideoSizeExceeded) {
		return &BusinessError{Info: "文件超过大小限制"}
	}
	if err != nil {
		return fmt.Errorf("write video chunk: %w", err)
	}
	if written <= 0 {
		_ = os.Remove(tempPath)
		return &BusinessError{Info: "视频分片不能为空"}
	}

	targetPath := filepath.Join(uploadDir, strconv.Itoa(input.ChunkIndex))
	backupPath, err := replaceChunkFile(tempPath, targetPath)
	if err != nil {
		return fmt.Errorf("replace video chunk: %w", err)
	}

	updatedState, err := scanUploadDiskState(uploadDir, info.Chunks)
	if err != nil {
		rollbackChunkReplacement(targetPath, backupPath)
		return fmt.Errorf("scan updated upload chunks: %w", err)
	}
	info.ChunkIndex = updatedState.HighestContiguous
	info.FileSize = updatedState.TotalSize
	updated, updateErr := s.store.UpdateIfExists(ctx, input.UserID, input.UploadID, info, uploadingFileTTL)
	if updateErr != nil {
		finishChunkReplacement(backupPath)
		return fmt.Errorf("update uploading file metadata: %w", updateErr)
	}
	if !updated {
		rollbackChunkReplacement(targetPath, backupPath)
		return &BusinessError{Info: "文件不存在，请重新上传"}
	}
	finishChunkReplacement(backupPath)
	return nil
}

func (s *VideoUploadService) uploadLock(userID string, uploadID string) *sync.Mutex {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(userID + ":" + uploadID))
	return &s.uploadLocks[int(hash.Sum32()%uint32(len(s.uploadLocks)))]
}

func (s *VideoUploadService) maxVideoSize(ctx context.Context) (int64, error) {
	setting := domain.DefaultSysSetting()
	if s.settingStore != nil {
		stored, exists, err := s.settingStore.Get(ctx)
		if err != nil {
			return 0, err
		}
		if exists {
			setting = stored
		}
	}
	setting = domain.NormalizeSysSetting(setting)
	return int64(setting.VideoSize) * bytesPerMB, nil
}

func (s *VideoUploadService) safeUploadDirectory(filePath string) (string, error) {
	cleanPath := filepath.Clean(filepath.FromSlash(strings.TrimSpace(filePath)))
	if cleanPath == "." || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") {
		return "", errors.New("invalid upload file path")
	}
	rootAbs, err := filepath.Abs(s.tempRoot)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, cleanPath))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
		return "", errors.New("upload path escapes temp root")
	}
	return targetAbs, nil
}

func scanUploadDiskState(uploadDir string, chunks int) (uploadDiskState, error) {
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return uploadDiskState{}, err
	}
	state := uploadDiskState{
		HighestContiguous: -1,
		ChunkSizes:        make(map[int]int64),
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		chunkIndex, err := strconv.Atoi(entry.Name())
		if err != nil || chunkIndex < 0 || chunkIndex >= chunks {
			continue
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return uploadDiskState{}, err
		}
		state.ChunkSizes[chunkIndex] = fileInfo.Size()
		state.TotalSize += fileInfo.Size()
	}
	for chunkIndex := 0; chunkIndex < chunks; chunkIndex++ {
		if _, exists := state.ChunkSizes[chunkIndex]; !exists {
			break
		}
		state.HighestContiguous = chunkIndex
	}
	return state, nil
}

func writeChunkToTemp(fileHeader *multipart.FileHeader, uploadDir string, chunkIndex int, maxBytes int64) (string, int64, error) {
	source, err := fileHeader.Open()
	if err != nil {
		return "", 0, err
	}
	defer source.Close()

	tempFile, err := os.CreateTemp(uploadDir, fmt.Sprintf(".chunk-%d-*.part", chunkIndex))
	if err != nil {
		return "", 0, err
	}
	tempPath := tempFile.Name()
	keepTemp := false
	defer func() {
		_ = tempFile.Close()
		if !keepTemp {
			_ = os.Remove(tempPath)
		}
	}()

	written, err := io.Copy(tempFile, io.LimitReader(source, maxBytes+1))
	if err != nil {
		return "", written, err
	}
	if written > maxBytes {
		return "", written, errVideoSizeExceeded
	}
	if err := tempFile.Sync(); err != nil {
		return "", written, err
	}
	if err := tempFile.Close(); err != nil {
		return "", written, err
	}
	keepTemp = true
	return tempPath, written, nil
}

func replaceChunkFile(tempPath string, targetPath string) (string, error) {
	backupPath := ""
	if _, err := os.Stat(targetPath); err == nil {
		backupFile, err := os.CreateTemp(filepath.Dir(targetPath), ".chunk-backup-*.part")
		if err != nil {
			_ = os.Remove(tempPath)
			return "", err
		}
		backupPath = backupFile.Name()
		if err := backupFile.Close(); err != nil {
			_ = os.Remove(tempPath)
			_ = os.Remove(backupPath)
			return "", err
		}
		if err := os.Remove(backupPath); err != nil {
			_ = os.Remove(tempPath)
			return "", err
		}
		if err := os.Rename(targetPath, backupPath); err != nil {
			_ = os.Remove(tempPath)
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tempPath)
		return "", err
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		if backupPath != "" {
			_ = os.Rename(backupPath, targetPath)
		}
		_ = os.Remove(tempPath)
		return "", err
	}
	return backupPath, nil
}

func rollbackChunkReplacement(targetPath string, backupPath string) {
	_ = os.Remove(targetPath)
	if backupPath != "" {
		_ = os.Rename(backupPath, targetPath)
	}
}

func finishChunkReplacement(backupPath string) {
	if backupPath != "" {
		_ = os.Remove(backupPath)
	}
}

func validUploadID(uploadID string) bool {
	if len(uploadID) != uploadIDLength {
		return false
	}
	for _, value := range uploadID {
		if !strings.ContainsRune(uploadIDAlphabet, value) {
			return false
		}
	}
	return true
}

func createUploadDirectory(uploadDir string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(uploadDir), 0755); err != nil {
		return false, err
	}
	if err := os.Mkdir(uploadDir, 0755); err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func cleanupUploadDirectory(uploadDir string, cause error) error {
	if err := os.RemoveAll(uploadDir); err != nil {
		return fmt.Errorf("%v; cleanup upload directory: %w", cause, err)
	}
	return cause
}

func randomUploadID(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	var builder strings.Builder
	builder.Grow(length)
	max := big.NewInt(int64(len(uploadIDAlphabet)))
	for index := 0; index < length; index++ {
		value, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		builder.WriteByte(uploadIDAlphabet[value.Int64()])
	}
	return builder.String(), nil
}

var errVideoSizeExceeded = errors.New("video size exceeded")
