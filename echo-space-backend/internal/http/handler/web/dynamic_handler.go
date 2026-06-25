package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type DynamicHandler struct {
	dynamicService *webservice.DynamicService
}

func NewDynamicHandler(dynamicService *webservice.DynamicService) *DynamicHandler {
	return &DynamicHandler{dynamicService: dynamicService}
}

func (h *DynamicHandler) LoadFollowUsers(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	result, err := h.dynamicService.LoadFollowUsers(c.Request.Context(), tokenUserInfo.UserID)
	if err != nil {
		handleDynamicError(c, "web dynamic load follow users", err)
		return
	}
	response.Success(c, result)
}

func (h *DynamicHandler) LoadFeed(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	result, err := h.dynamicService.LoadFeed(c.Request.Context(), webservice.LoadDynamicFeedInput{
		UserID:      tokenUserInfo.UserID,
		FocusUserID: formOrQuery(c, "focusUserId"),
		Cursor:      formOrQuery(c, "cursor"),
		PageSize:    parseWebIntWithDefault(formOrQuery(c, "pageSize"), 10),
	})
	if err != nil {
		handleDynamicError(c, "web dynamic load feed", err)
		return
	}
	response.Success(c, result)
}

func handleDynamicError(c *gin.Context, logPrefix string, err error) {
	if businessError, ok := webservice.IsBusinessError(err); ok {
		response.BusinessError(c, businessError.Info, nil)
		return
	}

	log.Printf("%s: %v", logPrefix, err)
	response.ServerError(c, nil)
}
