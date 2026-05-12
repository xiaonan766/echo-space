package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type AccountHandler struct {
	accountService *webservice.AccountService
}

func NewAccountHandler(accountService *webservice.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

func (h *AccountHandler) CheckCode(c *gin.Context) {
	result, err := h.accountService.GenerateCheckCode()
	if err != nil {
		log.Printf("generate web check code: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *AccountHandler) Register(c *gin.Context) {
	err := h.accountService.Register(c.Request.Context(), webservice.RegisterInput{
		Email:            c.PostForm("email"),
		NickName:         c.PostForm("nickName"),
		RegisterPassword: c.PostForm("registerPassword"),
		CheckCodeKey:     c.PostForm("checkCodeKey"),
		CheckCode:        c.PostForm("checkCode"),
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web register: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func (h *AccountHandler) Login(c *gin.Context) {
	oldToken, _ := c.Cookie(webservice.WebTokenCookieName)
	result, err := h.accountService.Login(c.Request.Context(), webservice.LoginInput{
		Email:        c.PostForm("email"),
		Password:     c.PostForm("password"),
		CheckCodeKey: c.PostForm("checkCodeKey"),
		CheckCode:    c.PostForm("checkCode"),
		OldToken:     oldToken,
		LoginIP:      c.ClientIP(),
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web login: %v", err)
		response.ServerError(c, nil)
		return
	}

	c.SetCookie(webservice.WebTokenCookieName, result.Token, 0, "/", "", false, false)
	response.Success(c, result)
}

func (h *AccountHandler) Logout(c *gin.Context) {
	token := c.GetString(middleware.ContextTokenKey)
	c.SetCookie(webservice.WebTokenCookieName, "", -1, "/", "", false, false)

	if err := h.accountService.Logout(c.Request.Context(), token); err != nil {
		log.Printf("web logout: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}
