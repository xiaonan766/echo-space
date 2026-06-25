package web

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type CommentHandler struct {
	service        *webservice.CommentService
	accountService *webservice.AccountService
}

func NewCommentHandler(service *webservice.CommentService, accountService ...*webservice.AccountService) *CommentHandler {
	handler := &CommentHandler{service: service}
	if len(accountService) > 0 {
		handler.accountService = accountService[0]
	}
	return handler
}

func (h *CommentHandler) LoadComment(c *gin.Context) {
	pCommentID, ok := parseCommentIntWithDefault(formOrQuery(c, "pCommentId"), 0)
	if !ok || pCommentID < 0 {
		response.BusinessError(c, "参数错误", nil)
		return
	}
	orderType, ok := parseCommentIntWithDefault(formOrQuery(c, "orderType"), 0)
	if !ok {
		response.BusinessError(c, "参数错误", nil)
		return
	}

	userID, err := h.optionalUserID(c)
	if err != nil {
		log.Printf("web load comment token: %v", err)
		response.ServerError(c, nil)
		return
	}

	result, err := h.service.LoadComment(c.Request.Context(), webservice.LoadCommentInput{
		VideoID:    formOrQuery(c, "videoId"),
		PCommentID: pCommentID,
		OrderType:  orderType,
		Cursor:     formOrQuery(c, "cursor"),
		UserID:     userID,
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web load comment: %v", err)
		response.ServerError(c, nil)
		return
	}
	if result == nil {
		response.Success(c, nil)
		return
	}
	response.Success(c, result)
}

func (h *CommentHandler) PostComment(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	replyCommentID, ok := parseOptionalCommentID(formOrQuery(c, "replyCommentId"))
	if !ok {
		response.BusinessError(c, "参数错误", nil)
		return
	}

	result, err := h.service.PostComment(c.Request.Context(), webservice.PostCommentInput{
		UserID:         tokenUserInfo.UserID,
		NickName:       tokenUserInfo.NickName,
		Avatar:         tokenUserInfo.Avatar,
		Content:        formOrQuery(c, "content"),
		ImgPath:        formOrQuery(c, "imgPath"),
		VideoID:        formOrQuery(c, "videoId"),
		ReplyCommentID: replyCommentID,
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web post comment: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *CommentHandler) optionalUserID(c *gin.Context) (string, error) {
	if h.accountService == nil {
		return "", nil
	}

	token := getOptionalWebToken(c)
	if token == "" {
		return "", nil
	}

	tokenUserInfo, ok, err := h.accountService.GetTokenUserInfo(c.Request.Context(), token)
	if err != nil {
		return "", err
	}
	if !ok || tokenUserInfo == nil {
		return "", nil
	}
	return tokenUserInfo.UserID, nil
}

func parseCommentIntWithDefault(value string, fallback int) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, true
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func parseOptionalCommentID(value string) (*int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return nil, false
	}
	return &parsed, true
}
