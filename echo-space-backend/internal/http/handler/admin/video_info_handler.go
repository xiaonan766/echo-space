package admin

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

type VideoInfoHandler struct {
	videoInfoService *adminservice.VideoInfoService
}

func NewVideoInfoHandler(videoInfoService *adminservice.VideoInfoService) *VideoInfoHandler {
	return &VideoInfoHandler{
		videoInfoService: videoInfoService,
	}
}

func (h *VideoInfoHandler) LoadVideoList(c *gin.Context) {
	status, ok := parseOptionalVideoFormStatus(formOrQuery(c, "status"))
	if !ok {
		response.BusinessError(c, "参数错误", nil)
		return
	}

	recommendType, ok := parseOptionalRecommendType(formOrQuery(c, "recommendType"))
	if !ok {
		response.BusinessError(c, "参数错误", nil)
		return
	}

	result, err := h.videoInfoService.LoadVideoList(c.Request.Context(), adminservice.VideoInfoListInput{
		PageNo:         parseIntWithDefault(formOrQuery(c, "pageNo"), 1),
		PageSize:       parseIntWithDefault(formOrQuery(c, "pageSize"), 15),
		VideoNameFuzzy: formOrQuery(c, "videoNameFuzzy"),
		PCategoryID:    parseIntWithDefault(formOrQuery(c, "pCategoryId"), 0),
		CategoryID:     parseIntWithDefault(formOrQuery(c, "categoryId"), 0),
		Status:         status,
		RecommendType:  recommendType,
	})
	if err != nil {
		log.Printf("admin load video list: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *VideoInfoHandler) LoadVideoPList(c *gin.Context) {
	result, err := h.videoInfoService.LoadVideoPList(c.Request.Context(), formOrQuery(c, "videoId"))
	if err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("admin load video p list: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *VideoInfoHandler) AuditVideo(c *gin.Context) {
	status, err := strconv.Atoi(strings.TrimSpace(formOrQuery(c, "status")))
	if err != nil {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	err = h.videoInfoService.AuditVideo(c.Request.Context(), adminservice.AuditVideoInput{
		VideoID: formOrQuery(c, "videoId"),
		Status:  status,
		Reason:  formOrQuery(c, "reason"),
	})
	if err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("admin audit video: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func formOrQuery(c *gin.Context, key string) string {
	if value := c.PostForm(key); strings.TrimSpace(value) != "" {
		return value
	}
	return c.Query(key)
}

func parseOptionalVideoFormStatus(value string) (*int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}

	status, err := strconv.Atoi(value)
	if err != nil || !adminservice.IsValidVideoPostStatusForAdmin(status) {
		return nil, false
	}
	return &status, true
}

func parseOptionalRecommendType(value string) (*int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}

	recommendType, err := strconv.Atoi(value)
	if err != nil || (recommendType != 0 && recommendType != 1) {
		return nil, false
	}
	return &recommendType, true
}
