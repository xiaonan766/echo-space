package web

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type StatisticsHandler struct {
	service *webservice.StatisticsService
}

func NewStatisticsHandler(service *webservice.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{service: service}
}

func (h *StatisticsHandler) GetActualTimeStatisticsInfo(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	result, err := h.service.GetActualTimeStatisticsInfo(c.Request.Context(), tokenUserInfo.UserID)
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web get actual time statistics info: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *StatisticsHandler) GetWeekStatisticsInfo(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	dataType, err := strconv.Atoi(strings.TrimSpace(formOrQuery(c, "dataType")))
	if err != nil {
		response.BusinessError(c, "参数错误", nil)
		return
	}

	result, err := h.service.GetWeekStatisticsInfo(c.Request.Context(), tokenUserInfo.UserID, dataType)
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web get week statistics info: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}
