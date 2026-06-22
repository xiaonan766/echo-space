package router

import (
	"github.com/gin-gonic/gin"

	filehandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler"
	webhandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler/web"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

func registerWebRoutes(group *gin.RouterGroup, fileGroup *gin.RouterGroup, deps Dependencies) {
	userRepository := repository.NewUserRepository(deps.DB)
	accountService := webservice.NewAccountService(deps.Cache, userRepository)
	accountHandler := webhandler.NewAccountHandler(accountService)
	uploadingFileStore := cache.NewUploadingFileStore(deps.Redis)
	sysSettingStore := cache.NewSysSettingStore(deps.Cache, "echo-space:sys_setting")
	videoUploadService := webservice.NewVideoUploadService(uploadingFileStore, sysSettingStore, deps.Config.File.ResourceRoot)
	videoUploadHandler := webhandler.NewVideoUploadHandler(videoUploadService)
	videoPostRepository := repository.NewVideoPostRepository(deps.DB)
	webFileHandler := filehandler.NewFileHandler(deps.Config.File)

	categoryRepository := repository.NewCategoryRepository(deps.DB)
	categoryService := webservice.NewCategoryService(categoryRepository)
	categoryHandler := webhandler.NewCategoryHandler(categoryService)
	videoPostService := webservice.NewVideoPostService(videoPostRepository, categoryRepository, uploadingFileStore, sysSettingStore, deps.VideoTranscodePublisher, deps.Config.File.ResourceRoot)
	videoPostHandler := webhandler.NewVideoPostHandler(videoPostService)
	videoRepository := repository.NewVideoRepository(deps.DB)
	videoService := webservice.NewVideoService(videoRepository)
	videoHandler := webhandler.NewVideoHandler(videoService)

	shopRepository := repository.NewShopRepository(deps.DB)
	shopRecommendStore := cache.NewShopRecommendStore(deps.Cache, deps.Redis)
	shopService := webservice.NewShopService(shopRepository, shopRecommendStore)
	shopHandler := webhandler.NewShopHandler(shopService)
	shopOrderRepository := repository.NewShopOrderRepository(deps.DB)
	shopStockStore := cache.NewShopStockStore(deps.Redis)
	shopOrderService := webservice.NewShopOrderService(shopRepository, shopOrderRepository, shopStockStore, deps.StockLockPublisher)
	shopOrderHandler := webhandler.NewShopOrderHandler(shopOrderService)

	accountGroup := group.Group("/account")
	accountGroup.GET("/checkCode", accountHandler.CheckCode)
	accountGroup.POST("/checkCode", accountHandler.CheckCode)
	accountGroup.POST("/register", accountHandler.Register)
	accountGroup.POST("/login", accountHandler.Login)
	accountGroup.POST("/logout", middleware.WebAuth(accountService), accountHandler.Logout)

	ucenterGroup := group.Group("/ucenter", middleware.WebAuth(accountService))
	ucenterGroup.POST("/postVideo", videoPostHandler.PostVideo)

	fileGroup.POST("/preUploadVideo", middleware.WebAuth(accountService), videoUploadHandler.PreUploadVideo)
	fileGroup.POST("/uploadVideo", middleware.WebAuth(accountService), videoUploadHandler.UploadVideo)
	fileGroup.POST("/uploadImage", middleware.WebAuth(accountService), webFileHandler.UploadImage)

	categoryGroup := group.Group("/category")
	categoryGroup.GET("/loadAllCategory", categoryHandler.LoadAllCategory)
	categoryGroup.POST("/loadAllCategory", categoryHandler.LoadAllCategory)

	videoGroup := group.Group("/video")
	videoGroup.GET("/loadVideo", videoHandler.LoadVideo)
	videoGroup.POST("/loadVideo", videoHandler.LoadVideo)

	shopGroup := group.Group("/shop")
	shopGroup.GET("/loadRecommend", shopHandler.LoadRecommend)
	shopGroup.POST("/loadRecommend", shopHandler.LoadRecommend)
	shopGroup.GET("/loadList", shopHandler.LoadList)
	shopGroup.POST("/loadList", shopHandler.LoadList)
	shopGroup.GET("/getPeripheralDetail", shopHandler.GetPeripheralDetail)
	shopGroup.POST("/getPeripheralDetail", shopHandler.GetPeripheralDetail)

	shopOrderGroup := shopGroup.Group("/order", middleware.WebAuth(accountService))
	shopOrderGroup.POST("/create", shopOrderHandler.CreateOrder)
	shopOrderGroup.GET("/detail", shopOrderHandler.GetOrderDetail)
	shopOrderGroup.POST("/detail", shopOrderHandler.GetOrderDetail)
	shopOrderGroup.GET("/loadOrder", shopOrderHandler.LoadOrder)
	shopOrderGroup.POST("/loadOrder", shopOrderHandler.LoadOrder)
}
