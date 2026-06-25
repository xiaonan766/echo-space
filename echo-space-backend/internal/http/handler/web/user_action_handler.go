package web

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type UserActionHandler struct {
	service *webservice.UserActionService
}

func NewUserActionHandler(service *webservice.UserActionService) *UserActionHandler {
	return &UserActionHandler{service: service}
}

func (h *UserActionHandler) DoAction(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	actionType, ok := parseRequiredActionInt(formOrQuery(c, "actionType"))
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}
	actionCount, ok := parseActionIntWithDefault(formOrQuery(c, "actionCount"), 1)
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}
	commentID, ok := parseActionIntWithDefault(formOrQuery(c, "commentId"), 0)
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	err := h.service.DoAction(c.Request.Context(), webservice.DoUserActionInput{
		UserID:      tokenUserInfo.UserID,
		VideoID:     formOrQuery(c, "videoId"),
		ActionType:  actionType,
		ActionCount: actionCount,
		CommentID:   commentID,
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web do user action: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func parseRequiredActionInt(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}

func parseActionIntWithDefault(value string, fallback int) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, true
	}

	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}
