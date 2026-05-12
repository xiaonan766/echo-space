package router

import (
	"github.com/gin-gonic/gin"

	adminhandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler/admin"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

func registerAdminRoutes(group *gin.RouterGroup, deps Dependencies) {
	healthHandler := adminhandler.NewHealthHandler(deps.Redis)
	accountService := adminservice.NewAccountService(deps.Cache, deps.Config.Admin)
	accountHandler := adminhandler.NewAccountHandler(accountService)
	indexHandler := adminhandler.NewIndexHandler()
	settingService := adminservice.NewSettingService(deps.Cache)
	settingHandler := adminhandler.NewSettingHandler(settingService)
	categoryRepository := repository.NewCategoryRepository(deps.DB)
	categoryService := adminservice.NewCategoryService(categoryRepository)
	categoryHandler := adminhandler.NewCategoryHandler(categoryService)
	userRepository := repository.NewUserRepository(deps.DB)
	userService := adminservice.NewUserService(userRepository)
	userHandler := adminhandler.NewUserHandler(userService)

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

	settingGroup := group.Group("/setting")
	settingGroup.GET("/getSetting", settingHandler.GetSetting)
	settingGroup.POST("/getSetting", settingHandler.GetSetting)
	settingGroup.POST("/saveSetting", settingHandler.SaveSetting)

	categoryGroup := group.Group("/category")
	categoryGroup.GET("/loadCategory", categoryHandler.LoadCategory)
	categoryGroup.POST("/loadCategory", categoryHandler.LoadCategory)
	categoryGroup.POST("/saveCategory", categoryHandler.SaveCategory)
	categoryGroup.POST("/delCategory", categoryHandler.DeleteCategory)
	categoryGroup.POST("/changeSort", categoryHandler.ChangeSort)

	userGroup := group.Group("/user")
	userGroup.GET("/loadUser", userHandler.LoadUser)
	userGroup.POST("/loadUser", userHandler.LoadUser)
	userGroup.POST("/changeStatus", userHandler.ChangeStatus)
}
