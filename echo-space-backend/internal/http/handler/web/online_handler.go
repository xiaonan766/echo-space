package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type OnlineHandler struct {
	service          *webservice.VideoOnlineService
	hotMetricService *webservice.VideoHotMetricService
}

func NewOnlineHandler(service *webservice.VideoOnlineService, hotMetricService ...*webservice.VideoHotMetricService) *OnlineHandler {
	handler := &OnlineHandler{service: service}
	if len(hotMetricService) > 0 {
		handler.hotMetricService = hotMetricService[0]
	}
	return handler
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

func (h *OnlineHandler) ReportVideoPlayHot(c *gin.Context) {
	if h.hotMetricService == nil {
		response.Success(c, nil)
		return
	}

	if err := h.hotMetricService.ReportVideoPlayHot(c.Request.Context(), webservice.ReportVideoPlayHotInput{
		VideoID:  formOrQuery(c, "videoId"),
		DeviceID: formOrQuery(c, "deviceId"),
		ClientIP: c.ClientIP(),
	}); err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web report video play hot: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}
