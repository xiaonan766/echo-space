package router

import (
	"github.com/gin-gonic/gin"

	filehandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler"
	adminhandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler/admin"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
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
	interactRepository := repository.NewInteractRepository(deps.DB)
	interactService := adminservice.NewInteractService(interactRepository)
	interactHandler := adminhandler.NewInteractHandler(interactService)
	shopRepository := repository.NewShopRepository(deps.DB)
	shopRecommendStore := cache.NewShopRecommendStore(deps.Cache)
	shopService := adminservice.NewShopService(shopRepository, shopRecommendStore)
	shopHandler := adminhandler.NewShopHandler(shopService)
	fileHandler := filehandler.NewFileHandler(deps.Config.File)

	group.GET("/health", healthHandler.Health)

	accountGroup := group.Group("/account")
	accountGroup.GET("/checkCode", accountHandler.CheckCode)
	accountGroup.POST("/checkCode", accountHandler.CheckCode)
	accountGroup.POST("/login", accountHandler.Login)

	filePublicGroup := group.Group("/file")
	filePublicGroup.GET("/getResource", fileHandler.GetResource)

	authGroup := group.Group("")
	authGroup.Use(middleware.AdminAuth(accountService))

	fileGroup := authGroup.Group("/file")
	fileGroup.POST("/uploadImage", fileHandler.UploadImage)

	indexGroup := authGroup.Group("/index")
	indexGroup.GET("/getActualTimeStatisticsInfo", indexHandler.GetActualTimeStatisticsInfo)
	indexGroup.POST("/getActualTimeStatisticsInfo", indexHandler.GetActualTimeStatisticsInfo)
	indexGroup.GET("/getWeekStatisticsInfo", indexHandler.GetWeekStatisticsInfo)
	indexGroup.POST("/getWeekStatisticsInfo", indexHandler.GetWeekStatisticsInfo)

	settingGroup := authGroup.Group("/setting")
	settingGroup.GET("/getSetting", settingHandler.GetSetting)
	settingGroup.POST("/getSetting", settingHandler.GetSetting)
	settingGroup.POST("/saveSetting", settingHandler.SaveSetting)

	categoryGroup := authGroup.Group("/category")
	categoryGroup.GET("/loadCategory", categoryHandler.LoadCategory)
	categoryGroup.POST("/loadCategory", categoryHandler.LoadCategory)
	categoryGroup.POST("/saveCategory", categoryHandler.SaveCategory)
	categoryGroup.POST("/delCategory", categoryHandler.DeleteCategory)
	categoryGroup.POST("/changeSort", categoryHandler.ChangeSort)

	userGroup := authGroup.Group("/user")
	userGroup.GET("/loadUser", userHandler.LoadUser)
	userGroup.POST("/loadUser", userHandler.LoadUser)
	userGroup.POST("/changeStatus", userHandler.ChangeStatus)

	interactGroup := authGroup.Group("/interact")
	interactGroup.GET("/loadComment", interactHandler.LoadComment)
	interactGroup.POST("/loadComment", interactHandler.LoadComment)
	interactGroup.POST("/delComment", interactHandler.DeleteComment)
	interactGroup.GET("/loadDanmu", interactHandler.LoadDanmu)
	interactGroup.POST("/loadDanmu", interactHandler.LoadDanmu)
	interactGroup.POST("/delDanmu", interactHandler.DeleteDanmu)

	peripheralGroup := authGroup.Group("/shop/peripheral")
	peripheralGroup.GET("/loadPeripheral", shopHandler.LoadPeripheral)
	peripheralGroup.POST("/loadPeripheral", shopHandler.LoadPeripheral)
	peripheralGroup.GET("/getPeripheral", shopHandler.GetPeripheral)
	peripheralGroup.POST("/getPeripheral", shopHandler.GetPeripheral)
	peripheralGroup.POST("/savePeripheral", shopHandler.SavePeripheral)
	peripheralGroup.POST("/changeStatus", shopHandler.ChangePeripheralStatus)
}
