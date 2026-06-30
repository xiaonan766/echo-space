package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type OnlineHandler struct {
	service *webservice.VideoOnlineService
}

func NewOnlineHandler(service *webservice.VideoOnlineService) *OnlineHandler {
	return &OnlineHandler{service: service}
}

func (h *OnlineHandler) ReportVideoPlayOnline(c *gin.Context) {
	count, err := h.service.ReportVideoPlayOnline(c.Request.Context(), webservice.ReportVideoPlayOnlineInput{
		FileID:   formOrQuery(c, "fileId"),
		DeviceID: formOrQuery(c, "deviceId"),
		ClientIP: c.ClientIP(),
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web report video play online: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, count)
}
