package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/mq"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	videoOutboxScanInterval     = 5 * time.Second
	videoOutboxBatchSize        = 20
	videoPublishedRetryAge      = 10 * time.Minute
	videoTranscodeLeaseDuration = 5 * time.Minute
	videoLeaseRefreshInterval   = time.Minute
	videoTranscodeMaxRetries    = 3
	mergedVideoFileName         = "source.mp4"
	transcodeResultFileName     = ".transcode-result.json"
	hlsWorkDirectoryName        = ".hls-work"
)

var videoTranscodeRetryDelays = [...]time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}

type VideoTranscodeService struct {
	repository  *repository.VideoPostRepository
	uploadStore *cache.UploadingFileStore
	publisher   VideoTranscodePublisher
	processor   *FFmpegVideoProcessor
	now         func() time.Time
}

func NewVideoTranscodeService(repo *repository.VideoPostRepository, uploadStore *cache.UploadingFileStore, publisher VideoTranscodePublisher, resourceRoot string) *VideoTranscodeService {
	return &VideoTranscodeService{
		repository: repo, uploadStore: uploadStore, publisher: publisher,
		processor: NewFFmpegVideoProcessor(resourceRoot), now: time.Now,
	}
}

func (s *VideoTranscodeService) StartOutboxPublisher(ctx context.Context) {
	go func() {
		s.publishPendingMessages(ctx)
		ticker := time.NewTicker(videoOutboxScanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.publishPendingMessages(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *VideoTranscodeService) HandleVideoTranscodeMessage(ctx context.Context, incoming mq.VideoTranscodeMessage) error {
	if s == nil || s.repository == nil || s.processor == nil {
		return errors.New("video transcode service is not ready")
	}
	lockToken, err := randomUploadID(videoMessageIDLength)
	if err != nil {
		return err
	}
	record, claimed, err := s.repository.ClaimTranscodeMessage(ctx, incoming.MessageID, lockToken, s.now().Add(videoTranscodeLeaseDuration))
	if err != nil || !claimed {
		return err
	}

	var message mq.VideoTranscodeMessage
	if err := json.Unmarshal([]byte(record.Payload), &message); err != nil {
		return s.persistTranscodeFailure(ctx, record.MessageID, record.FileID, lockToken, record.RetryCount, fmt.Errorf("decode transcode payload: %w", err))
	}
	if message.MessageID != record.MessageID || message.FileID != record.FileID || message.VideoID != record.VideoID || message.UserID != record.UserID || message.UploadID != record.UploadID {
		return s.persistTranscodeFailure(ctx, record.MessageID, record.FileID, lockToken, record.RetryCount, errors.New("transcode payload does not match outbox record"))
	}

	heartbeatCtx, stopHeartbeat := context.WithCancel(context.Background())
	defer stopHeartbeat()
	processCtx, cancelProcess := context.WithCancel(ctx)
	defer cancelProcess()
	go s.refreshLease(heartbeatCtx, message.MessageID, lockToken, cancelProcess)

	result, err := s.processor.Transcode(processCtx, message)
	if err != nil {
		return s.persistTranscodeFailure(ctx, message.MessageID, message.FileID, lockToken, record.RetryCount, err)
	}
	if err := s.repository.CompleteTranscodeMessage(ctx, message.MessageID, message.FileID, lockToken, repository.VideoTranscodeResult{
		FileSize: result.FileSize, FilePath: result.FilePath, Duration: result.Duration,
	}); errors.Is(err, repository.ErrVideoTranscodeClaimLost) {
		return nil
	} else if err != nil {
		return err
	}
	if err := s.processor.Cleanup(message); err != nil {
		log.Printf("cleanup video transcode source failed: messageID=%s err=%v", message.MessageID, err)
	}
	if s.uploadStore != nil {
		if err := s.uploadStore.Delete(context.Background(), message.UserID, message.UploadID); err != nil {
			log.Printf("delete uploading video metadata failed: uploadID=%s err=%v", message.UploadID, err)
		}
	}
	return nil
}

func (s *VideoTranscodeService) publishPendingMessages(ctx context.Context) {
	if s == nil || s.repository == nil || s.publisher == nil {
		return
	}
	publishedBefore := s.now().Add(-videoPublishedRetryAge)
	records, err := s.repository.ListTranscodeMessagesForPublish(ctx, videoOutboxBatchSize, publishedBefore)
	if err != nil {
		log.Printf("list video transcode outbox failed: %v", err)
		return
	}
	for _, record := range records {
		var message mq.VideoTranscodeMessage
		if err := json.Unmarshal([]byte(record.Payload), &message); err != nil {
			_, _ = s.repository.RetryOrFailTranscodeMessage(ctx, record.MessageID, record.FileID, "", 1, s.now(), err)
			continue
		}
		if err := s.publisher.PublishVideoTranscodeMessage(ctx, message); err != nil {
			_ = s.repository.DelayTranscodeMessagePublish(ctx, record.MessageID, s.now().Add(videoOutboxScanInterval), err)
			log.Printf("publish video transcode message failed, will retry: messageID=%s err=%v", record.MessageID, err)
			break
		}
		if err := s.repository.MarkTranscodeMessagePublished(ctx, record.MessageID, publishedBefore); err != nil {
			log.Printf("mark video transcode message published failed: messageID=%s err=%v", record.MessageID, err)
		}
	}
}

func (s *VideoTranscodeService) persistTranscodeFailure(ctx context.Context, messageID string, fileID string, lockToken string, retryCount int, cause error) error {
	delayIndex := retryCount
	if delayIndex >= len(videoTranscodeRetryDelays) {
		delayIndex = len(videoTranscodeRetryDelays) - 1
	}
	dead, err := s.repository.RetryOrFailTranscodeMessage(ctx, messageID, fileID, lockToken, videoTranscodeMaxRetries, s.now().Add(videoTranscodeRetryDelays[delayIndex]), cause)
	if errors.Is(err, repository.ErrVideoTranscodeClaimLost) {
		return nil
	}
	if err != nil {
		return err
	}
	if dead {
		log.Printf("video transcode task became dead: messageID=%s fileID=%s err=%v", messageID, fileID, cause)
	} else {
		log.Printf("video transcode task scheduled for retry: messageID=%s fileID=%s err=%v", messageID, fileID, cause)
	}
	return nil
}

func (s *VideoTranscodeService) refreshLease(ctx context.Context, messageID string, lockToken string, cancelProcess context.CancelFunc) {
	ticker := time.NewTicker(videoLeaseRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.repository.RefreshTranscodeLease(context.Background(), messageID, lockToken, s.now().Add(videoTranscodeLeaseDuration)); err != nil {
				log.Printf("refresh video transcode lease failed: messageID=%s err=%v", messageID, err)
				if errors.Is(err, repository.ErrVideoTranscodeClaimLost) {
					cancelProcess()
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

type FFmpegVideoProcessor struct {
	resourceRoot string
	runCommand   func(context.Context, string, ...string) ([]byte, error)
}

type FFmpegVideoResult struct {
	FileSize int64  `json:"fileSize"`
	FilePath string `json:"filePath"`
	Duration int    `json:"duration"`
}

func NewFFmpegVideoProcessor(resourceRoot string) *FFmpegVideoProcessor {
	resourceRoot = strings.TrimSpace(resourceRoot)
	if resourceRoot == "" {
		resourceRoot = defaultResourceRoot
	}
	return &FFmpegVideoProcessor{resourceRoot: resourceRoot, runCommand: runExternalCommand}
}

func (p *FFmpegVideoProcessor) Transcode(ctx context.Context, message mq.VideoTranscodeMessage) (FFmpegVideoResult, error) {
	targetDir, err := p.prepareTargetDirectory(message)
	if err != nil {
		return FFmpegVideoResult{}, err
	}
	if result, ok := readTranscodeResult(targetDir); ok {
		if _, err := os.Stat(filepath.Join(targetDir, "index.m3u8")); err == nil {
			return result, nil
		}
	}

	state, err := scanUploadDiskState(targetDir, message.Chunks)
	if err != nil {
		return FFmpegVideoResult{}, fmt.Errorf("scan video chunks: %w", err)
	}
	if state.HighestContiguous != message.Chunks-1 || len(state.ChunkSizes) != message.Chunks {
		return FFmpegVideoResult{}, errors.New("video chunks are incomplete")
	}

	mergedPath := filepath.Join(targetDir, mergedVideoFileName)
	if err := mergeVideoChunks(targetDir, mergedPath, message.Chunks); err != nil {
		return FFmpegVideoResult{}, err
	}
	mergedInfo, err := os.Stat(mergedPath)
	if err != nil {
		return FFmpegVideoResult{}, err
	}
	duration, err := p.probeDuration(ctx, mergedPath)
	if err != nil {
		return FFmpegVideoResult{}, err
	}
	if err := p.createHLS(ctx, mergedPath, targetDir); err != nil {
		return FFmpegVideoResult{}, err
	}
	result := FFmpegVideoResult{
		FileSize: mergedInfo.Size(),
		FilePath: filepath.ToSlash(filepath.Join("video", message.FilePath)),
		Duration: duration,
	}
	if err := writeTranscodeResult(targetDir, result); err != nil {
		return FFmpegVideoResult{}, err
	}
	return result, nil
}

func (p *FFmpegVideoProcessor) Cleanup(message mq.VideoTranscodeMessage) error {
	targetDir, err := safeResourceSubdirectory(filepath.Join(p.resourceRoot, "video"), message.FilePath)
	if err != nil {
		return err
	}
	for index := 0; index < message.Chunks; index++ {
		if err := os.Remove(filepath.Join(targetDir, strconv.Itoa(index))); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, name := range []string{mergedVideoFileName, transcodeResultFileName} {
		if err := os.Remove(filepath.Join(targetDir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (p *FFmpegVideoProcessor) prepareTargetDirectory(message mq.VideoTranscodeMessage) (string, error) {
	if message.Chunks <= 0 || !validUploadID(message.UploadID) {
		return "", errors.New("invalid video transcode message")
	}
	tempDir, err := safeResourceSubdirectory(filepath.Join(p.resourceRoot, temporaryVideoDir), message.FilePath)
	if err != nil {
		return "", err
	}
	targetDir, err := safeResourceSubdirectory(filepath.Join(p.resourceRoot, "video"), message.FilePath)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(targetDir); err == nil {
		state, scanErr := scanUploadDiskState(targetDir, message.Chunks)
		if scanErr == nil && len(state.ChunkSizes) > 0 {
			return targetDir, nil
		}
		if _, tempErr := os.Stat(tempDir); tempErr == nil {
			if removeErr := os.Remove(targetDir); removeErr != nil {
				return "", fmt.Errorf("remove empty video target directory: %w", removeErr)
			}
		} else {
			return targetDir, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if _, err := os.Stat(tempDir); err != nil {
		return "", fmt.Errorf("find temporary video directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return "", err
	}
	if err := os.Rename(tempDir, targetDir); err != nil {
		return "", fmt.Errorf("move video chunks to target directory: %w", err)
	}
	return targetDir, nil
}

func (p *FFmpegVideoProcessor) probeDuration(ctx context.Context, mergedPath string) (int, error) {
	output, err := p.runCommand(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", mergedPath)
	if err != nil {
		return 0, fmt.Errorf("ffprobe video duration: %w: %s", err, trimCommandOutput(output))
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("parse video duration %q", strings.TrimSpace(string(output)))
	}
	return int(math.Ceil(seconds)), nil
}

func (p *FFmpegVideoProcessor) createHLS(ctx context.Context, mergedPath string, targetDir string) error {
	workDir := filepath.Join(targetDir, hlsWorkDirectoryName)
	if err := os.RemoveAll(workDir); err != nil {
		return err
	}
	if err := os.Mkdir(workDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	playlistPath := filepath.Join(workDir, "index.m3u8")
	segmentPattern := filepath.Join(workDir, "segment_%05d.ts")
	output, err := p.runCommand(ctx, "ffmpeg",
		"-y", "-i", mergedPath,
		"-map", "0:v:0", "-map", "0:a?",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "23",
		"-c:a", "aac", "-b:a", "128k",
		"-hls_time", "10", "-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	)
	if err != nil {
		return fmt.Errorf("ffmpeg convert video to hls: %w: %s", err, trimCommandOutput(output))
	}
	return commitHLSFiles(workDir, targetDir)
}

func mergeVideoChunks(dir string, target string, chunks int) error {
	if _, err := os.Stat(target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tempFile, err := os.CreateTemp(dir, ".merged-*.part")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	keep := false
	defer func() {
		_ = tempFile.Close()
		if !keep {
			_ = os.Remove(tempPath)
		}
	}()
	for index := 0; index < chunks; index++ {
		chunk, err := os.Open(filepath.Join(dir, strconv.Itoa(index)))
		if err != nil {
			return fmt.Errorf("open video chunk %d: %w", index, err)
		}
		_, copyErr := io.Copy(tempFile, chunk)
		closeErr := chunk.Close()
		if copyErr != nil {
			return fmt.Errorf("merge video chunk %d: %w", index, copyErr)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, target); err != nil {
		return err
	}
	keep = true
	return nil
}

func commitHLSFiles(workDir string, targetDir string) error {
	oldSegments, _ := filepath.Glob(filepath.Join(targetDir, "segment_*.ts"))
	for _, old := range append(oldSegments, filepath.Join(targetDir, "index.m3u8")) {
		if err := os.Remove(old); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := os.Rename(filepath.Join(workDir, entry.Name()), filepath.Join(targetDir, entry.Name())); err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(targetDir, "index.m3u8")); err != nil {
		return errors.New("hls playlist was not generated")
	}
	return nil
}

func readTranscodeResult(targetDir string) (FFmpegVideoResult, bool) {
	content, err := os.ReadFile(filepath.Join(targetDir, transcodeResultFileName))
	if err != nil {
		return FFmpegVideoResult{}, false
	}
	var result FFmpegVideoResult
	if err := json.Unmarshal(content, &result); err != nil || result.FileSize <= 0 || result.Duration <= 0 {
		return FFmpegVideoResult{}, false
	}
	return result, true
}

func writeTranscodeResult(targetDir string, result FFmpegVideoResult) error {
	content, err := json.Marshal(result)
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(targetDir, ".transcode-result-*.part")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(targetDir, transcodeResultFileName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tempPath, filepath.Join(targetDir, transcodeResultFileName))
}

func runExternalCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func trimCommandOutput(output []byte) string {
	value := strings.TrimSpace(string(output))
	if len(value) > 2000 {
		return value[len(value)-2000:]
	}
	return value
}
