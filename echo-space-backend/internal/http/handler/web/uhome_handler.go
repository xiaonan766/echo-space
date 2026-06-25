package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type UhomeHandler struct {
	uhomeService *webservice.UhomeService
}

func NewUhomeHandler(uhomeService *webservice.UhomeService) *UhomeHandler {
	return &UhomeHandler{
		uhomeService: uhomeService,
	}
}

func (h *UhomeHandler) Focus(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	if err := h.uhomeService.FocusUser(c.Request.Context(), tokenUserInfo.UserID, formOrQuery(c, "focusUserId")); err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web uhome focus: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}
