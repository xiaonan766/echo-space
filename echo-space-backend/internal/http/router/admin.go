package router

import (
	"github.com/gin-gonic/gin"

	adminhandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler/admin"
)

func registerAdminRoutes(group *gin.RouterGroup, deps Dependencies) {
	healthHandler := adminhandler.NewHealthHandler(deps.Redis)

	group.GET("/health", healthHandler.Health)
	group.Group("/account")
}
