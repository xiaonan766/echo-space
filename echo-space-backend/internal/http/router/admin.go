package router

import (
	"github.com/gin-gonic/gin"

	adminhandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler/admin"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

func registerAdminRoutes(group *gin.RouterGroup, deps Dependencies) {
	healthHandler := adminhandler.NewHealthHandler(deps.Redis)
	accountService := adminservice.NewAccountService(deps.Redis, deps.Config.Admin)
	accountHandler := adminhandler.NewAccountHandler(accountService)
	indexHandler := adminhandler.NewIndexHandler()

	group.GET("/health", healthHandler.Health)

	accountGroup := group.Group("/account")
	accountGroup.GET("/checkCode", accountHandler.CheckCode)
	accountGroup.POST("/checkCode", accountHandler.CheckCode)
	accountGroup.POST("/login", accountHandler.Login)

	indexGroup := group.Group("/index")
	indexGroup.GET("/getActualTimeStatisticsInfo", indexHandler.GetActualTimeStatisticsInfo)
	indexGroup.POST("/getActualTimeStatisticsInfo", indexHandler.GetActualTimeStatisticsInfo)
	indexGroup.GET("/getWeekStatisticsInfo", indexHandler.GetWeekStatisticsInfo)
	indexGroup.POST("/getWeekStatisticsInfo", indexHandler.GetWeekStatisticsInfo)
}
