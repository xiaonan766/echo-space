package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type VideoHandler struct {
	videoService *webservice.VideoService
}

func NewVideoHandler(videoService *webservice.VideoService) *VideoHandler {
	return &VideoHandler{videoService: videoService}
}

func (h *VideoHandler) LoadVideo(c *gin.Context) {
	result, err := h.videoService.LoadVideo(c.Request.Context(), webservice.VideoListInput{
		PageNo:      parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		PageSize:    parseWebIntWithDefault(formOrQuery(c, "pageSize"), 15),
		PCategoryID: parseWebIntWithDefault(formOrQuery(c, "pCategoryId"), 0),
		CategoryID:  parseWebIntWithDefault(formOrQuery(c, "categoryId"), 0),
	})
	if err != nil {
		log.Printf("web load video: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}
