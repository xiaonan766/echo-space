package web

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type VideoUploadHandler struct {
	uploadService *webservice.VideoUploadService
}

func NewVideoUploadHandler(uploadService *webservice.VideoUploadService) *VideoUploadHandler {
	return &VideoUploadHandler{uploadService: uploadService}
}

func (h *VideoUploadHandler) PreUploadVideo(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	chunks, err := strconv.Atoi(strings.TrimSpace(c.PostForm("chunks")))
	if err != nil {
		response.BusinessError(c, "请输入正确的文件分片数量", nil)
		return
	}

	uploadID, err := h.uploadService.PreUploadVideo(c.Request.Context(), webservice.PreUploadVideoInput{
		UserID:   tokenUserInfo.UserID,
		FileName: c.PostForm("fileName"),
		Chunks:   chunks,
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web pre upload video: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, uploadID)
}

func (h *VideoUploadHandler) UploadVideo(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	chunkIndex, err := strconv.Atoi(strings.TrimSpace(c.PostForm("chunkIndex")))
	if err != nil {
		response.BusinessError(c, "请输入正确的分片索引", nil)
		return
	}
	chunkFile, err := c.FormFile("chunkFile")
	if err != nil {
		response.BusinessError(c, "请选择要上传的视频分片", nil)
		return
	}

	err = h.uploadService.UploadVideoChunk(c.Request.Context(), webservice.UploadVideoChunkInput{
		UserID:     tokenUserInfo.UserID,
		UploadID:   c.PostForm("uploadId"),
		ChunkIndex: chunkIndex,
		ChunkFile:  chunkFile,
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web upload video chunk: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}
