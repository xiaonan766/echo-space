package web

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateVideoPostInput(t *testing.T) {
	categoryID := 62
	valid := SaveVideoPostInput{
		UserID: "5171840855", VideoCover: "images/202606/cover.png", VideoName: "测试视频",
		PCategoryID: 60, CategoryID: &categoryID, PostType: 0, Tags: "测试,视频", DownloadPermission: 1,
		UploadFileList: []VideoPostUploadFile{{UploadID: "Abc123Def456Ghi", FileName: "第一集"}},
	}
	if err := validateVideoPostInput(valid); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*SaveVideoPostInput)
	}{
		{name: "missing user", mutate: func(input *SaveVideoPostInput) { input.UserID = "" }},
		{name: "unsafe cover", mutate: func(input *SaveVideoPostInput) { input.VideoCover = "../cover.png" }},
		{name: "long title", mutate: func(input *SaveVideoPostInput) { input.VideoName = strings.Repeat("题", 101) }},
		{name: "invalid post type", mutate: func(input *SaveVideoPostInput) { input.PostType = 2 }},
		{name: "repost without origin", mutate: func(input *SaveVideoPostInput) { input.PostType = 1 }},
		{name: "invalid interaction", mutate: func(input *SaveVideoPostInput) { input.Interaction = "0,2" }},
		{name: "invalid download permission", mutate: func(input *SaveVideoPostInput) { input.DownloadPermission = 2 }},
		{name: "empty file list", mutate: func(input *SaveVideoPostInput) { input.UploadFileList = nil }},
		{name: "invalid existing file id", mutate: func(input *SaveVideoPostInput) {
			input.UploadFileList = []VideoPostUploadFile{{FileID: "short", FileName: "第一集"}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			input.UploadFileList = append([]VideoPostUploadFile(nil), valid.UploadFileList...)
			test.mutate(&input)
			if err := validateVideoPostInput(input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBuildVideoPostKeepsDownloadPermission(t *testing.T) {
	input := SaveVideoPostInput{
		UserID: "5171840855", VideoCover: "images/202606/cover.png", VideoName: "测试视频",
		PCategoryID: 60, PostType: 0, Tags: "测试,视频", DownloadPermission: 0,
	}

	post := buildVideoPost(input, "Abc123Def4", time.Now(), 0)
	if post.DownloadPermission != 0 {
		t.Fatalf("DownloadPermission = %d, want 0", post.DownloadPermission)
	}
}

func TestSafeResourceSubdirectoryRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := safeResourceSubdirectory(root, "../escape"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	target, err := safeResourceSubdirectory(root, "20260622/user-upload")
	if err != nil {
		t.Fatalf("safe path returned error: %v", err)
	}
	want := filepath.Join(root, "20260622", "user-upload")
	if target != want {
		t.Fatalf("target = %q, want %q", target, want)
	}
}
