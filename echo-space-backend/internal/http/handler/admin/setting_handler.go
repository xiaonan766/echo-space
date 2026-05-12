package admin

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

type SettingHandler struct {
	settingService *adminservice.SettingService
}

func NewSettingHandler(settingService *adminservice.SettingService) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
	}
}

func (h *SettingHandler) GetSetting(c *gin.Context) {
	result, err := h.settingService.GetSetting(c.Request.Context())
	if err != nil {
		log.Printf("admin get setting: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *SettingHandler) SaveSetting(c *gin.Context) {
	var req domain.SysSetting
	if err := c.ShouldBind(&req); err != nil {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	if err := h.settingService.SaveSetting(c.Request.Context(), req); err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin save setting: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}
