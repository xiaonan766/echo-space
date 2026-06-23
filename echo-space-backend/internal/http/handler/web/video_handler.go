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

func (h *VideoHandler) LoadRecommendVideo(c *gin.Context) {
	result, err := h.videoService.LoadRecommendVideo(c.Request.Context())
	if err != nil {
		log.Printf("web load recommend video: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *VideoHandler) LoadVideoPList(c *gin.Context) {
	result, err := h.videoService.LoadVideoPList(c.Request.Context(), formOrQuery(c, "videoId"))
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web load video p list: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}
