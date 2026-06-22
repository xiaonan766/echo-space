package web

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/mq"
)

const transcodeTestUploadID = "Abc123Def456Ghi"

func TestFFmpegVideoProcessorTranscodeAndResume(t *testing.T) {
	root := t.TempDir()
	message := prepareTranscodeChunks(t, root, []byte("first"), []byte("second"))
	processor := NewFFmpegVideoProcessor(root)
	commandCount := 0
	processor.runCommand = func(_ context.Context, name string, args ...string) ([]byte, error) {
		commandCount++
		switch name {
		case "ffprobe":
			return []byte("2.25\n"), nil
		case "ffmpeg":
			playlist := args[len(args)-1]
			if err := os.WriteFile(playlist, []byte("#EXTM3U\nsegment_00000.ts\n"), 0644); err != nil {
				return nil, err
			}
			return nil, os.WriteFile(filepath.Join(filepath.Dir(playlist), "segment_00000.ts"), []byte("segment"), 0644)
		default:
			return nil, errors.New("unexpected command")
		}
	}

	result, err := processor.Transcode(context.Background(), message)
	if err != nil {
		t.Fatalf("Transcode() error = %v", err)
	}
	if result.Duration != 3 || result.FileSize != int64(len("firstsecond")) {
		t.Fatalf("result = %+v", result)
	}
	if result.FilePath != "video/20260622/user"+transcodeTestUploadID {
		t.Fatalf("filePath = %q", result.FilePath)
	}
	targetDir := filepath.Join(root, "video", "20260622", "user"+transcodeTestUploadID)
	assertPathExists(t, filepath.Join(targetDir, "index.m3u8"))
	assertPathExists(t, filepath.Join(targetDir, "segment_00000.ts"))

	processor.runCommand = func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("completed transcode should use result marker")
	}
	if _, err := processor.Transcode(context.Background(), message); err != nil {
		t.Fatalf("resume Transcode() error = %v", err)
	}
	if commandCount != 2 {
		t.Fatalf("command count = %d, want 2", commandCount)
	}
	if err := processor.Cleanup(message); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "0")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("chunk was not removed: %v", err)
	}
	assertPathExists(t, filepath.Join(targetDir, "index.m3u8"))
}

func TestFFmpegVideoProcessorRejectsIncompleteChunks(t *testing.T) {
	root := t.TempDir()
	message := prepareTranscodeChunks(t, root, []byte("only-first"))
	message.Chunks = 2
	processor := NewFFmpegVideoProcessor(root)
	if _, err := processor.Transcode(context.Background(), message); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("Transcode() error = %v, want incomplete chunks", err)
	}
}

func TestFFmpegVideoProcessorRealFFmpeg(t *testing.T) {
	if os.Getenv("RUN_FFMPEG_INTEGRATION") != "1" {
		t.Skip("set RUN_FFMPEG_INTEGRATION=1 to run the real ffmpeg test")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed")
	}

	root := t.TempDir()
	source := filepath.Join(root, "input.mp4")
	output, err := runExternalCommand(context.Background(), "ffmpeg",
		"-y", "-f", "lavfi", "-i", "color=c=blue:s=320x180:d=1",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", source,
	)
	if err != nil {
		t.Fatalf("generate test video: %v: %s", err, trimCommandOutput(output))
	}
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	middle := len(content) / 2
	message := prepareTranscodeChunks(t, root, content[:middle], content[middle:])
	processor := NewFFmpegVideoProcessor(root)
	result, err := processor.Transcode(context.Background(), message)
	if err != nil {
		t.Fatalf("real Transcode() error = %v", err)
	}
	if result.Duration <= 0 || result.FileSize != int64(len(content)) {
		t.Fatalf("real result = %+v", result)
	}
	playlist := filepath.Join(root, "video", "20260622", "user"+transcodeTestUploadID, "index.m3u8")
	playlistContent, err := os.ReadFile(playlist)
	if err != nil || !strings.Contains(string(playlistContent), "#EXTM3U") {
		t.Fatalf("invalid playlist: err=%v content=%q", err, string(playlistContent))
	}
}

func prepareTranscodeChunks(t *testing.T, root string, chunks ...[]byte) mq.VideoTranscodeMessage {
	t.Helper()
	relative := filepath.ToSlash(filepath.Join("20260622", "user"+transcodeTestUploadID))
	dir := filepath.Join(root, "temp", filepath.FromSlash(relative))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for index, chunk := range chunks {
		if err := os.WriteFile(filepath.Join(dir, string(rune('0'+index))), chunk, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return mq.VideoTranscodeMessage{
		MessageID: "Message1234567890Message123456789", FileID: "File1234567890File123", VideoID: "Video12345",
		UserID: "user", UploadID: transcodeTestUploadID, FilePath: relative, FileName: "test.mp4", Chunks: len(chunks),
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}
