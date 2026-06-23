package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

var allowedImageExts = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".bmp":  {},
	".webp": {},
}

type FileHandler struct {
	resourceRoot        string
	maxImageSize        int64
	videoPostRepository *repository.VideoPostRepository
}

func NewFileHandler(fileConfig config.FileConfig, videoPostRepository ...*repository.VideoPostRepository) *FileHandler {
	maxImageMB := fileConfig.MaxImageMB
	if maxImageMB <= 0 {
		maxImageMB = 10
	}

	resourceRoot := strings.TrimSpace(fileConfig.ResourceRoot)
	if resourceRoot == "" {
		resourceRoot = "resources"
	}

	var videoRepo *repository.VideoPostRepository
	if len(videoPostRepository) > 0 {
		videoRepo = videoPostRepository[0]
	}

	return &FileHandler{
		resourceRoot:        resourceRoot,
		maxImageSize:        int64(maxImageMB) * 1024 * 1024,
		videoPostRepository: videoRepo,
	}
}

func (h *FileHandler) UploadImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxImageSize)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BusinessError(c, "\u8bf7\u9009\u62e9\u8981\u4e0a\u4f20\u7684\u56fe\u7247", nil)
		return
	}

	sourceName, err := h.saveImage(fileHeader)
	if err != nil {
		if errors.Is(err, errInvalidImageExt) {
			response.BusinessError(c, "\u4ec5\u652f\u6301 jpg\u3001jpeg\u3001png\u3001gif\u3001bmp\u3001webp \u683c\u5f0f\u56fe\u7247", nil)
			return
		}
		if errors.Is(err, errFileTooLarge) {
			response.BusinessError(c, "\u56fe\u7247\u5927\u5c0f\u8d85\u51fa\u9650\u5236", nil)
			return
		}

		log.Printf("upload image: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, sourceName)
}

func (h *FileHandler) GetResource(c *gin.Context) {
	sourceName := strings.TrimSpace(c.Query("sourceName"))
	if sourceName == "" {
		response.NotFound(c)
		return
	}

	targetPath, ok := h.safeResourcePath(sourceName)
	if !ok {
		response.NotFound(c)
		return
	}

	if _, err := os.Stat(targetPath); err != nil {
		response.NotFound(c)
		return
	}

	c.File(targetPath)
}

func (h *FileHandler) GetVideoResource(c *gin.Context) {
	h.serveVideoResource(c, "index.m3u8")
}

func (h *FileHandler) GetVideoResourceSegment(c *gin.Context) {
	resourceName := strings.TrimSpace(c.Param("resourceName"))
	h.serveVideoResource(c, resourceName)
}

func (h *FileHandler) saveImage(fileHeader *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if _, ok := allowedImageExts[ext]; !ok {
		return "", errInvalidImageExt
	}
	if fileHeader.Size > h.maxImageSize {
		return "", errFileTooLarge
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	now := time.Now()
	dirName := filepath.Join("images", now.Format("200601"))
	targetDir := filepath.Join(h.resourceRoot, dirName)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	fileName, err := randomFileName(ext)
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(targetDir, fileName)
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, file); err != nil {
		return "", err
	}

	return filepath.ToSlash(filepath.Join(dirName, fileName)), nil
}

func (h *FileHandler) serveVideoResource(c *gin.Context, resourceName string) {
	if h.videoPostRepository == nil {
		response.NotFound(c)
		return
	}

	fileID := strings.TrimSpace(c.Param("fileId"))
	if !validVideoFileID(fileID) {
		response.NotFound(c)
		return
	}

	resourceName, ok := normalizeVideoResourceName(resourceName)
	if !ok {
		response.NotFound(c)
		return
	}

	videoFile, err := h.videoPostRepository.FindPostFileByFileID(c.Request.Context(), fileID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.NotFound(c)
		return
	}
	if err != nil {
		log.Printf("load video resource file info: fileID=%s err=%v", fileID, err)
		response.ServerError(c, nil)
		return
	}
	if strings.TrimSpace(videoFile.FilePath) == "" {
		response.NotFound(c)
		return
	}

	targetPath, ok := h.safeResourcePath(path.Join(videoFile.FilePath, resourceName))
	if !ok {
		response.NotFound(c)
		return
	}
	if _, err := os.Stat(targetPath); err != nil {
		response.NotFound(c)
		return
	}

	switch strings.ToLower(filepath.Ext(resourceName)) {
	case ".m3u8":
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
	case ".ts":
		c.Header("Content-Type", "video/mp2t")
	}
	c.File(targetPath)
}

func (h *FileHandler) safeResourcePath(sourceName string) (string, bool) {
	cleanName := filepath.Clean(strings.TrimLeft(sourceName, `/\`))
	if cleanName == "." || strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
		return "", false
	}

	rootAbs, err := filepath.Abs(h.resourceRoot)
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

func normalizeVideoResourceName(resourceName string) (string, bool) {
	resourceName = strings.Trim(strings.TrimSpace(resourceName), `/\`)
	if resourceName == "" {
		resourceName = "index.m3u8"
	}
	if resourceName != path.Base(resourceName) || strings.Contains(resourceName, `\`) {
		return "", false
	}

	lowerName := strings.ToLower(resourceName)
	if lowerName == "index.m3u8" || (strings.HasPrefix(lowerName, "segment_") && strings.HasSuffix(lowerName, ".ts")) {
		return resourceName, true
	}
	return "", false
}

func validVideoFileID(fileID string) bool {
	if len(fileID) != 20 {
		return false
	}
	for _, char := range fileID {
		if char >= '0' && char <= '9' {
			continue
		}
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		return false
	}
	return true
}

func randomFileName(ext string) (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return time.Now().Format("20060102150405") + "_" + hex.EncodeToString(buffer) + ext, nil
}

var (
	errInvalidImageExt = errors.New("invalid image extension")
	errFileTooLarge    = errors.New("file too large")
)
