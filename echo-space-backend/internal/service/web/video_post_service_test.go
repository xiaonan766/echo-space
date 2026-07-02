package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
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

func TestValidateImagePostInput(t *testing.T) {
	valid := normalizeVideoPostInput(SaveVideoPostInput{
		UserID: "5171840855", VideoCover: "images/202607/cover.png", VideoName: "测试图片投稿", PCategoryID: 60, PostType: 0,
		ContentType: domain.ContentTypeImage, Tags: "图片,测试", DownloadPermission: 1,
		ImageList: []ImagePostUploadFile{{SourceName: "images/202607/image.png", FileName: "第一张"}},
	})
	if err := validateVideoPostInput(valid); err != nil {
		t.Fatalf("valid image input returned error: %v", err)
	}

	missingCover := valid
	missingCover.VideoCover = ""
	if err := validateVideoPostInput(missingCover); err == nil {
		t.Fatal("expected missing image cover validation error")
	}

	badCoverExt := valid
	badCoverExt.VideoCover = "images/202607/cover.txt"
	if err := validateVideoPostInput(badCoverExt); err == nil {
		t.Fatal("expected invalid image cover extension validation error")
	}

	tooMany := valid
	tooMany.ImageList = make([]ImagePostUploadFile, maxImagePostCount+1)
	for index := range tooMany.ImageList {
		tooMany.ImageList[index] = ImagePostUploadFile{
			SourceName: "images/202607/image" + string(rune('a'+index)) + ".png",
			FileName:   "图片",
		}
	}
	if err := validateVideoPostInput(tooMany); err == nil {
		t.Fatal("expected too many image validation error")
	}

	unsafe := valid
	unsafe.VideoCover = "../image.png"
	unsafe.ImageList = []ImagePostUploadFile{{SourceName: "../image.png", FileName: "图片"}}
	if err := validateVideoPostInput(unsafe); err == nil {
		t.Fatal("expected unsafe image validation error")
	}
}

func TestValidateImageCoverChecksResource(t *testing.T) {
	resourceRoot := t.TempDir()
	sourceName := "images/202607/cover.png"
	targetPath := filepath.Join(resourceRoot, filepath.FromSlash(sourceName))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		t.Fatalf("mkdir cover directory: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("cover"), 0644); err != nil {
		t.Fatalf("write cover file: %v", err)
	}

	service := &VideoPostService{resourceRoot: resourceRoot}
	if err := service.validateImageCover(sourceName); err != nil {
		t.Fatalf("valid cover returned error: %v", err)
	}
	if err := service.validateImageCover("images/202607/missing.png"); err == nil {
		t.Fatal("expected missing cover file validation error")
	}
	if err := service.validateImageCover("../cover.png"); err == nil {
		t.Fatal("expected unsafe cover path validation error")
	}
}

func TestBuildImageFilesGeneratesUniqueUploadIDs(t *testing.T) {
	resourceRoot := t.TempDir()
	sourceNames := []string{
		"images/202607/first.png",
		"images/202607/second.jpg",
	}
	for _, sourceName := range sourceNames {
		targetPath := filepath.Join(resourceRoot, filepath.FromSlash(sourceName))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			t.Fatalf("mkdir image directory: %v", err)
		}
		if err := os.WriteFile(targetPath, []byte("image"), 0644); err != nil {
			t.Fatalf("write image file: %v", err)
		}
	}

	generatedCount := 0
	service := &VideoPostService{
		resourceRoot: resourceRoot,
		generateID: func(length int) (string, error) {
			generatedCount++
			return strings.Repeat(string(rune('A'+generatedCount)), length), nil
		},
	}
	files, err := service.buildImageFiles("5171840855", "Abc123Def4", []ImagePostUploadFile{
		{SourceName: sourceNames[0], FileName: "第一张"},
		{SourceName: sourceNames[1], FileName: "第二张"},
	}, 1)
	if err != nil {
		t.Fatalf("buildImageFiles returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].UploadID == "" || files[1].UploadID == "" {
		t.Fatal("image file upload id should not be empty")
	}
	if files[0].UploadID == files[1].UploadID {
		t.Fatal("image file upload id should be unique per image")
	}
	if files[0].TransferResult != domain.VideoFileTransferSuccess || files[1].TransferResult != domain.VideoFileTransferSuccess {
		t.Fatal("image files should be stored as transfer success")
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
