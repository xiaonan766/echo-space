package web

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type UhomeHandler struct {
	uhomeService   *webservice.UhomeService
	accountService *webservice.AccountService
}

func NewUhomeHandler(uhomeService *webservice.UhomeService, accountService ...*webservice.AccountService) *UhomeHandler {
	handler := &UhomeHandler{
		uhomeService: uhomeService,
	}
	if len(accountService) > 0 {
		handler.accountService = accountService[0]
	}
	return handler
}

func (h *UhomeHandler) GetUserInfo(c *gin.Context) {
	currentUserID, err := h.optionalUserID(c)
	if err != nil {
		log.Printf("web uhome get user info token: %v", err)
		response.ServerError(c, nil)
		return
	}

	result, err := h.uhomeService.GetUserInfo(c.Request.Context(), currentUserID, formOrQuery(c, "userId"))
	if err != nil {
		if notFoundError, ok := webservice.IsNotFoundError(err); ok {
			response.Error(c, response.CodeNotFound, notFoundError.Info, nil)
			return
		}
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web uhome get user info: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *UhomeHandler) UpdateUserInfo(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	sex, err := strconv.Atoi(formOrQuery(c, "sex"))
	if err != nil {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	updatedUser, err := h.uhomeService.UpdateUserInfo(c.Request.Context(), tokenUserInfo.UserID, webservice.UpdateUserInfoInput{
		UserID:             formOrQuery(c, "userId"),
		Avatar:             formOrQuery(c, "avatar"),
		NickName:           formOrQuery(c, "nickName"),
		Sex:                sex,
		Birthday:           formOrQuery(c, "birthday"),
		School:             formOrQuery(c, "school"),
		PersonIntroduction: formOrQuery(c, "personIntroduction"),
		NoticeInfo:         formOrQuery(c, "noticeInfo"),
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web uhome update user info: %v", err)
		response.ServerError(c, nil)
		return
	}

	if h.accountService != nil {
		token := c.GetString(middleware.ContextTokenKey)
		if err := h.accountService.UpdateTokenUserInfo(c.Request.Context(), token, updatedUser); err != nil {
			log.Printf("web uhome update token info: %v", err)
			response.ServerError(c, nil)
			return
		}
	}

	response.Success(c, nil)
}

func (h *UhomeHandler) Focus(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	if err := h.uhomeService.FocusUser(c.Request.Context(), tokenUserInfo.UserID, formOrQuery(c, "focusUserId")); err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web uhome focus: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func (h *UhomeHandler) optionalUserID(c *gin.Context) (string, error) {
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
