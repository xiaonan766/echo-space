package admin

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

type AccountHandler struct {
	accountService *adminservice.AccountService
}

type loginRequest struct {
	Account      string `form:"account" binding:"required"`
	Password     string `form:"password" binding:"required"`
	CheckCodeKey string `form:"checkCodeKey" binding:"required"`
	CheckCode    string `form:"checkCode" binding:"required"`
}

func NewAccountHandler(accountService *adminservice.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

func (h *AccountHandler) CheckCode(c *gin.Context) {
	result, err := h.accountService.GenerateCheckCode()
	if err != nil {
		log.Printf("generate admin check code: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *AccountHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBind(&req); err != nil {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	oldToken, _ := c.Cookie(adminservice.AdminTokenCookieName)
	result, err := h.accountService.Login(c.Request.Context(), adminservice.LoginInput{
		Account:      req.Account,
		Password:     req.Password,
		CheckCodeKey: req.CheckCodeKey,
		CheckCode:    req.CheckCode,
		OldToken:     oldToken,
		LoginIP:      c.ClientIP(),
	})
	if err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin login: %v", err)
		response.ServerError(c, nil)
		return
	}

	c.SetCookie(adminservice.AdminTokenCookieName, result.Token, -1, "/", "", false, false)
	response.Success(c, result.Account)
}
