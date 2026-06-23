package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type SysSettingHandler struct {
	service *webservice.SysSettingService
}

func NewSysSettingHandler(service *webservice.SysSettingService) *SysSettingHandler {
	return &SysSettingHandler{service: service}
}

func (h *SysSettingHandler) GetSetting(c *gin.Context) {
	result, err := h.service.GetSetting(c.Request.Context())
	if err != nil {
		log.Printf("web get sys setting: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}
