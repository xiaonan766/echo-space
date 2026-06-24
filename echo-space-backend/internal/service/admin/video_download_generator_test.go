package admin

import (
	"strings"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

func TestBuildDownloadTailTextDoesNotContainUserID(t *testing.T) {
	text := buildDownloadTailText(repository.VideoDownloadGenerationJob{
		VideoID:   "Abc123Def4",
		UserID:    "5171840855",
		NickName:  "测试作者",
		VideoName: "测试视频",
	})

	for _, want := range []string{"Echo Space", "作者：测试作者", "视频ID：Abc123Def4"} {
		if !strings.Contains(text, want) {
			t.Fatalf("tail text %q does not contain %q", text, want)
		}
	}
	if strings.Contains(text, "5171840855") || strings.Contains(text, "作者ID") {
		t.Fatalf("tail text should not contain author id: %q", text)
	}
}

func TestSafeDownloadResourcePathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, ok := safeDownloadResourcePath(root, "../escape/download.mp4"); ok {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, ok := safeDownloadResourcePath(root, "video/202606/test/download.mp4"); !ok {
		t.Fatal("expected normal resource path to be accepted")
	}
}

func TestEscapeFFmpegFilterPathEscapesWindowsDriveColon(t *testing.T) {
	got := escapeFFmpegFilterPath(`D:\workspace-teach\echo-space\tail.txt`)
	wantPrefix := `D\\:/workspace-teach/echo-space/tail.txt`
	if got != wantPrefix {
		t.Fatalf("escaped path = %q, want %q", got, wantPrefix)
	}
}
