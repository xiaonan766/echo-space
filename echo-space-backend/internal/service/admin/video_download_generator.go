package admin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	downloadVideoFileName    = "download.mp4"
	downloadTailTextFileName = ".download-tail.txt"
	downloadGenerateTimeout  = 30 * time.Minute
)

type VideoDownloadGenerator struct {
	repository   *repository.VideoPostRepository
	resourceRoot string
	fontPath     string
	runCommand   func(context.Context, string, ...string) ([]byte, error)
}

func NewVideoDownloadGenerator(repo *repository.VideoPostRepository, resourceRoot string) *VideoDownloadGenerator {
	resourceRoot = strings.TrimSpace(resourceRoot)
	if resourceRoot == "" {
		resourceRoot = "resources"
	}
	return &VideoDownloadGenerator{
		repository:   repo,
		resourceRoot: resourceRoot,
		fontPath:     findVideoTailFont(),
		runCommand:   runDownloadCommand,
	}
}

func (g *VideoDownloadGenerator) GenerateAsync(videoID string) {
	if g == nil || g.repository == nil {
		return
	}
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), downloadGenerateTimeout)
		defer cancel()
		g.generate(ctx, videoID)
	}()
}

func (g *VideoDownloadGenerator) generate(ctx context.Context, videoID string) {
	jobs, err := g.repository.ListDownloadGenerationJobs(ctx, videoID)
	if err != nil {
		log.Printf("list video download generation jobs failed: videoID=%s err=%v", videoID, err)
		return
	}
	for _, job := range jobs {
		if err := g.generateOne(ctx, job); err != nil {
			log.Printf("generate video download file failed: videoID=%s fileID=%s err=%v", job.VideoID, job.FileID, err)
			if markErr := g.repository.MarkVideoFileDownloadFailed(context.Background(), job.FileID); markErr != nil {
				log.Printf("mark video download failed status failed: fileID=%s err=%v", job.FileID, markErr)
			}
		}
	}
}

func (g *VideoDownloadGenerator) generateOne(ctx context.Context, job repository.VideoDownloadGenerationJob) error {
	if strings.TrimSpace(job.FilePath) == "" {
		return errors.New("video file path is empty")
	}
	if g.fontPath == "" {
		return errors.New("video tail font is not available")
	}

	if err := g.repository.MarkVideoFileDownloadGenerating(context.Background(), job.FileID); err != nil {
		return err
	}

	sourcePlaylist, ok := safeDownloadResourcePath(g.resourceRoot, filepath.ToSlash(filepath.Join(job.FilePath, "index.m3u8")))
	if !ok {
		return errors.New("video playlist path is invalid")
	}
	if _, err := os.Stat(sourcePlaylist); err != nil {
		return fmt.Errorf("find video playlist: %w", err)
	}

	outputResourcePath := filepath.ToSlash(filepath.Join(job.FilePath, downloadVideoFileName))
	outputPath, ok := safeDownloadResourcePath(g.resourceRoot, outputResourcePath)
	if !ok {
		return errors.New("download output path is invalid")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	tailTextPath := filepath.Join(filepath.Dir(outputPath), downloadTailTextFileName)
	if err := os.WriteFile(tailTextPath, []byte(buildDownloadTailText(job)), 0644); err != nil {
		return err
	}
	defer os.Remove(tailTextPath)

	tempOutput, err := os.CreateTemp(filepath.Dir(outputPath), ".download-*.mp4")
	if err != nil {
		return err
	}
	tempOutputPath := tempOutput.Name()
	if err := tempOutput.Close(); err != nil {
		_ = os.Remove(tempOutputPath)
		return err
	}
	defer os.Remove(tempOutputPath)

	output, err := g.runCommand(ctx, "ffmpeg", g.downloadCommandArgs(sourcePlaylist, tailTextPath, tempOutputPath)...)
	if err != nil {
		return fmt.Errorf("ffmpeg generate download video: %w: %s", err, trimDownloadCommandOutput(output))
	}

	if err := os.Remove(outputPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(tempOutputPath, outputPath); err != nil {
		return err
	}
	return g.repository.MarkVideoFileDownloadSuccess(context.Background(), job.FileID, outputResourcePath)
}

func (g *VideoDownloadGenerator) downloadCommandArgs(sourcePlaylist string, tailTextPath string, outputPath string) []string {
	tailFilter := fmt.Sprintf(
		"drawtext=fontfile=%s:textfile=%s:fontcolor=white:fontsize=42:x=(w-text_w)/2:y=(h-text_h)/2:line_spacing=18",
		escapeFFmpegFilterPath(g.fontPath),
		escapeFFmpegFilterPath(tailTextPath),
	)
	return []string{
		"-y",
		"-i", sourcePlaylist,
		"-f", "lavfi", "-t", "3", "-i", "color=c=black:s=1280x720:r=30",
		"-filter_complex",
		fmt.Sprintf("[0:v]scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=30,format=yuv420p[v0];[1:v]%s,format=yuv420p[v1];[v0][v1]concat=n=2:v=1:a=0[v]", tailFilter),
		"-map", "[v]",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	}
}

func buildDownloadTailText(job repository.VideoDownloadGenerationJob) string {
	nickName := strings.TrimSpace(job.NickName)
	if nickName == "" {
		nickName = "未知作者"
	}
	return strings.Join([]string{
		"Echo Space",
		"作者：" + nickName,
		"视频ID：" + strings.TrimSpace(job.VideoID),
	}, "\n")
}

func safeDownloadResourcePath(root string, sourceName string) (string, bool) {
	cleanName := filepath.Clean(strings.TrimLeft(sourceName, `/\`))
	if cleanName == "." || strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
		return "", false
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, cleanName))
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return targetAbs, true
}

func findVideoTailFont() string {
	candidates := []string{
		`C:\Windows\Fonts\msyh.ttc`,
		`C:\Windows\Fonts\simhei.ttf`,
		`C:\Windows\Fonts\simsun.ttc`,
		`C:\Windows\Fonts\arial.ttf`,
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func escapeFFmpegFilterPath(path string) string {
	value := filepath.ToSlash(path)
	value = strings.ReplaceAll(value, `\`, `/`)
	value = strings.ReplaceAll(value, `:`, `\\:`)
	value = strings.ReplaceAll(value, `'`, `\\'`)
	value = strings.ReplaceAll(value, `,`, `\\,`)
	return value
}

func runDownloadCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func trimDownloadCommandOutput(output []byte) string {
	value := strings.TrimSpace(string(output))
	if len(value) > 2000 {
		return value[len(value)-2000:]
	}
	return value
}
