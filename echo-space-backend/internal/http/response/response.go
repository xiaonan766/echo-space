package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	StatusSuccess = "success"
	StatusError   = "error"
)

const (
	CodeSuccess      = 200
	CodeNotFound     = 404
	CodeServerError  = 500
	CodeBusinessFail = 600
	CodeLoginTimeout = 901
)

const (
	InfoSuccess      = "\u8bf7\u6c42\u6210\u529f"
	InfoNotFound     = "\u8bf7\u6c42\u5730\u5740\u4e0d\u5b58\u5728"
	InfoServerError  = "\u670d\u52a1\u5668\u9519\u8bef"
	InfoBusinessFail = "\u4e1a\u52a1\u5f02\u5e38"
	InfoLoginTimeout = "\u767b\u5f55\u8d85\u65f6"
)

type VO struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
	Info   string `json:"info"`
	Data   any    `json:"data"`
}

func Success(c *gin.Context, data any) {
	write(c, StatusSuccess, CodeSuccess, InfoSuccess, data)
}

func BusinessError(c *gin.Context, info string, data any) {
	if info == "" {
		info = InfoBusinessFail
	}
	write(c, StatusError, CodeBusinessFail, info, data)
}

func Error(c *gin.Context, code int, info string, data any) {
	if info == "" {
		info = InfoBusinessFail
	}
	write(c, StatusError, code, info, data)
}

func ServerError(c *gin.Context, data any) {
	write(c, StatusError, CodeServerError, InfoServerError, data)
}

func NotFound(c *gin.Context) {
	write(c, StatusError, CodeNotFound, InfoNotFound, nil)
}

func LoginTimeout(c *gin.Context) {
	write(c, StatusError, CodeLoginTimeout, InfoLoginTimeout, nil)
}

func write(c *gin.Context, status string, code int, info string, data any) {
	c.JSON(http.StatusOK, VO{
		Status: status,
		Code:   code,
		Info:   info,
		Data:   data,
	})
}
