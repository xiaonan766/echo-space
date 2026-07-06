package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type UcenterContentHandler struct {
	service *webservice.UcenterContentService
}

func NewUcenterContentHandler(service *webservice.UcenterContentService) *UcenterContentHandler {
	return &UcenterContentHandler{service: service}
}

func (h *UcenterContentHandler) LoadAllVideo(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	result, err := h.service.LoadAllVideo(c.Request.Context(), tokenUserInfo.UserID)
	if err != nil {
		handleUcenterContentError(c, "web load all ucenter video", err)
		return
	}
	response.Success(c, result)
}

func (h *UcenterContentHandler) LoadComment(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	result, err := h.service.LoadComment(c.Request.Context(), webservice.UcenterInteractListInput{
		UserID:   tokenUserInfo.UserID,
		VideoID:  formOrQuery(c, "videoId"),
		Cursor:   formOrQuery(c, "cursor"),
		PageSize: parseWebIntWithDefault(formOrQuery(c, "pageSize"), 15),
	})
	if err != nil {
		handleUcenterContentError(c, "web load ucenter comment", err)
		return
	}
	response.Success(c, result)
}

func (h *UcenterContentHandler) LoadDanmu(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	result, err := h.service.LoadDanmu(c.Request.Context(), webservice.UcenterInteractListInput{
		UserID:   tokenUserInfo.UserID,
		VideoID:  formOrQuery(c, "videoId"),
		Cursor:   formOrQuery(c, "cursor"),
		PageSize: parseWebIntWithDefault(formOrQuery(c, "pageSize"), 15),
	})
	if err != nil {
		handleUcenterContentError(c, "web load ucenter danmu", err)
		return
	}
	response.Success(c, result)
}

func (h *UcenterContentHandler) LoadUserCollection(c *gin.Context) {
	result, err := h.service.LoadUserCollection(c.Request.Context(), webservice.UhomeCollectionListInput{
		UserID:   formOrQuery(c, "userId"),
		PageNo:   parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		PageSize: parseWebIntWithDefault(formOrQuery(c, "pageSize"), 15),
	})
	if err != nil {
		handleUcenterContentError(c, "web load uhome user collection", err)
		return
	}
	response.Success(c, result)
}

func handleUcenterContentError(c *gin.Context, logPrefix string, err error) {
	if businessError, ok := webservice.IsBusinessError(err); ok {
		response.BusinessError(c, businessError.Info, nil)
		return
	}

	log.Printf("%s: %v", logPrefix, err)
	response.ServerError(c, nil)
}
