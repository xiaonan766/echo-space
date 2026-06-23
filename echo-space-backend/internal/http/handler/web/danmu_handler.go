package web

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type DanmuHandler struct {
	service *webservice.DanmuService
}

func NewDanmuHandler(service *webservice.DanmuService) *DanmuHandler {
	return &DanmuHandler{service: service}
}

func (h *DanmuHandler) PostDanmu(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	mode, err := strconv.Atoi(formOrQuery(c, "mode"))
	if err != nil {
		response.BusinessError(c, "参数错误", nil)
		return
	}
	danmuTime, err := strconv.Atoi(formOrQuery(c, "time"))
	if err != nil {
		response.BusinessError(c, "参数错误", nil)
		return
	}

	err = h.service.PostDanmu(c.Request.Context(), webservice.PostDanmuInput{
		UserID:   tokenUserInfo.UserID,
		Text:     formOrQuery(c, "text"),
		Mode:     mode,
		Color:    formOrQuery(c, "color"),
		Time:     danmuTime,
		FileID:   formOrQuery(c, "fileId"),
		VideoID:  formOrQuery(c, "videoId"),
		ClientIP: c.ClientIP(),
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web post danmu: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}
