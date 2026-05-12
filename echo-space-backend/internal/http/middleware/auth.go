package middleware

import (
	"context"
	"log"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

const (
	ContextTokenKey            = "authToken"
	ContextAdminTokenInfoKey   = "adminTokenInfo"
	ContextWebTokenUserInfoKey = "webTokenUserInfo"
)

type tokenLoader func(ctx context.Context, token string) (any, bool, error)

func AdminAuth(accountService *adminservice.AccountService) gin.HandlerFunc {
	return tokenAuth(adminservice.AdminTokenCookieName, ContextAdminTokenInfoKey, func(ctx context.Context, token string) (any, bool, error) {
		return accountService.GetTokenInfo(ctx, token)
	})
}

func WebAuth(accountService *webservice.AccountService) gin.HandlerFunc {
	return tokenAuth(webservice.WebTokenCookieName, ContextWebTokenUserInfoKey, func(ctx context.Context, token string) (any, bool, error) {
		return accountService.GetTokenUserInfo(ctx, token)
	})
}

func tokenAuth(cookieName string, contextUserKey string, loadToken tokenLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := getTokenFromRequest(c, cookieName)
		if token == "" {
			response.LoginTimeout(c)
			c.Abort()
			return
		}

		tokenInfo, ok, err := loadToken(c.Request.Context(), token)
		if err != nil {
			log.Printf("load login token failed: %v", err)
			response.ServerError(c, nil)
			c.Abort()
			return
		}
		if !ok {
			response.LoginTimeout(c)
			c.Abort()
			return
		}

		c.Set(ContextTokenKey, token)
		c.Set(contextUserKey, tokenInfo)
		c.Next()
	}
}

func getTokenFromRequest(c *gin.Context, name string) string {
	if token, err := c.Cookie(name); err == nil {
		if token = normalizeToken(token); token != "" {
			return token
		}
	}
	return normalizeToken(c.GetHeader(name))
}

func normalizeToken(token string) string {
	token = strings.TrimSpace(token)
	if strings.EqualFold(token, "null") || strings.EqualFold(token, "undefined") {
		return ""
	}
	return token
}
