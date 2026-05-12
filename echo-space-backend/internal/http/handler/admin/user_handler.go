package admin

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

type UserHandler struct {
	userService *adminservice.UserService
}

func NewUserHandler(userService *adminservice.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) LoadUser(c *gin.Context) {
	status, ok := parseOptionalStatus(c.PostForm("status"))
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	result, err := h.userService.LoadUser(c.Request.Context(), adminservice.UserListInput{
		PageNo:        parseIntWithDefault(c.PostForm("pageNo"), 1),
		PageSize:      parseIntWithDefault(c.PostForm("pageSize"), 15),
		NickNameFuzzy: c.PostForm("nickNameFuzzy"),
		Status:        status,
	})
	if err != nil {
		log.Printf("admin load user: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *UserHandler) ChangeStatus(c *gin.Context) {
	userID := c.PostForm("userId")
	status, ok := parseRequiredStatus(c.PostForm("status"))
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	if err := h.userService.ChangeStatus(c.Request.Context(), userID, status); err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin change user status: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func parseIntWithDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseOptionalStatus(value string) (*int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}

	status, ok := parseRequiredStatus(value)
	if !ok {
		return nil, false
	}
	return &status, true
}

func parseRequiredStatus(value string) (int, bool) {
	status, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	if status != 0 && status != 1 {
		return 0, false
	}
	return status, true
}
