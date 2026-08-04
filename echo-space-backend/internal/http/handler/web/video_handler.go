package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type VideoHandler struct {
	videoService      *webservice.VideoService
	accountService    *webservice.AccountService
	hotRankingService *webservice.VideoHotRankingService
}

func NewVideoHandler(videoService *webservice.VideoService, accountService ...*webservice.AccountService) *VideoHandler {
	handler := &VideoHandler{videoService: videoService}
	if len(accountService) > 0 {
		handler.accountService = accountService[0]
	}
	return handler
}

func NewVideoHandlerWithHot(videoService *webservice.VideoService, hotRankingService *webservice.VideoHotRankingService, accountService ...*webservice.AccountService) *VideoHandler {
	handler := NewVideoHandler(videoService, accountService...)
	handler.hotRankingService = hotRankingService
	return handler
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

func (h *VideoHandler) Search(c *gin.Context) {
	result, err := h.videoService.SearchVideo(c.Request.Context(), webservice.VideoSearchInput{
		Keyword:   formOrQuery(c, "keyword"),
		OrderType: parseOptionalWebInt(formOrQuery(c, "orderType")),
		PageNo:    parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		PageSize:  parseWebIntWithDefault(formOrQuery(c, "pageSize"), 30),
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web search video: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *VideoHandler) GetVideoInfo(c *gin.Context) {
	userID, err := h.optionalUserID(c)
	if err != nil {
		log.Printf("web get video info token: %v", err)
		response.ServerError(c, nil)
		return
	}

	result, err := h.videoService.GetVideoInfo(c.Request.Context(), formOrQuery(c, "videoId"), userID)
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web get video info: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *VideoHandler) optionalUserID(c *gin.Context) (string, error) {
	if h.accountService == nil {
		return "", nil
	}

	token := getOptionalWebToken(c)
	if token == "" {
		return "", nil
	}

	tokenUserInfo, ok, err := h.accountService.GetTokenUserInfo(c.Request.Context(), token)
	if err != nil {
		return "", err
	}
	if !ok || tokenUserInfo == nil {
		return "", nil
	}
	return tokenUserInfo.UserID, nil
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

func (h *VideoHandler) LoadHotVideoList(c *gin.Context) {
	pageNo := parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1)
	pageSize := parseWebIntWithDefault(formOrQuery(c, "pageSize"), 20)
	if h.hotRankingService == nil {
		response.Success(c, domain.NewPaginationResult([]domain.WebHotVideoItem{}, 0, pageNo, pageSize))
		return
	}

	result, err := h.hotRankingService.LoadHotVideoList(c.Request.Context(), webservice.HotVideoListInput{
		PageNo:   pageNo,
		PageSize: pageSize,
	})
	if err != nil {
		log.Printf("web load hot video list: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}
