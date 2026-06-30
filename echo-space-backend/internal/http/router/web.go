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

func registerWebRoutes(group *gin.RouterGroup, fileGroup *gin.RouterGroup, interactGroup *gin.RouterGroup, deps Dependencies) {
	userRepository := repository.NewUserRepository(deps.DB)
	dynamicActiveStore := cache.NewDynamicActiveStore(deps.Redis)
	accountService := webservice.NewAccountService(deps.Cache, userRepository, dynamicActiveStore)
	accountHandler := webhandler.NewAccountHandler(accountService)
	uhomeService := webservice.NewUhomeService(userRepository)
	uhomeHandler := webhandler.NewUhomeHandler(uhomeService, accountService)
	peripheralCabinetRepository := repository.NewShopCabinetRepository(deps.DB)
	peripheralCabinetService := webservice.NewPeripheralCabinetService(peripheralCabinetRepository)
	peripheralCabinetHandler := webhandler.NewPeripheralCabinetHandler(peripheralCabinetService, accountService)
	uploadingFileStore := cache.NewUploadingFileStore(deps.Redis)
	sysSettingStore := cache.NewSysSettingStore(deps.Cache, "echo-space:sys_setting")
	videoUploadService := webservice.NewVideoUploadService(uploadingFileStore, sysSettingStore, deps.Config.File.ResourceRoot)
	videoUploadHandler := webhandler.NewVideoUploadHandler(videoUploadService)
	videoPostRepository := repository.NewVideoPostRepository(deps.DB)
	videoRepository := repository.NewVideoRepository(deps.DB)
	webFileHandler := filehandler.NewPublicFileHandler(deps.Config.File, videoRepository)

	categoryRepository := repository.NewCategoryRepository(deps.DB)
	categoryService := webservice.NewCategoryService(categoryRepository)
	categoryHandler := webhandler.NewCategoryHandler(categoryService)
	videoPostService := webservice.NewVideoPostService(videoPostRepository, categoryRepository, uploadingFileStore, sysSettingStore, deps.VideoTranscodePublisher, deps.Config.File.ResourceRoot)
	videoPostHandler := webhandler.NewVideoPostHandler(videoPostService)
	videoService := webservice.NewVideoService(
		videoRepository,
		webservice.WithVideoSearch(deps.VideoSearch),
		webservice.WithSearchKeywordStore(deps.SearchKeywordStore),
	)
	videoHandler := webhandler.NewVideoHandler(videoService, accountService)
	dynamicRepository := repository.NewDynamicRepository(deps.DB)
	dynamicService := webservice.NewDynamicService(dynamicRepository, dynamicActiveStore)
	dynamicHandler := webhandler.NewDynamicHandler(dynamicService)
	sysSettingService := webservice.NewSysSettingService(sysSettingStore)
	sysSettingHandler := webhandler.NewSysSettingHandler(sysSettingService)
	interactRepository := repository.NewInteractRepository(deps.DB)
	danmuLimiter := cache.NewDanmuRateLimiter(deps.Redis)
	danmuService := webservice.NewDanmuService(interactRepository, sysSettingStore, danmuLimiter)
	danmuHandler := webhandler.NewDanmuHandler(danmuService)
	commentService := webservice.NewCommentService(interactRepository, deps.Config.CommentReview)
	commentHandler := webhandler.NewCommentHandler(commentService, accountService)
	userActionService := webservice.NewUserActionService(interactRepository)
	userActionHandler := webhandler.NewUserActionHandler(userActionService)
	videoOnlineStore := cache.NewVideoOnlineStore(deps.Redis)
	videoOnlineService := webservice.NewVideoOnlineService(videoOnlineStore)
	onlineHandler := webhandler.NewOnlineHandler(videoOnlineService)
	statisticsRepository := repository.NewStatisticsRepository(deps.DB)
	statisticsService := webservice.NewStatisticsService(statisticsRepository)
	statisticsHandler := webhandler.NewStatisticsHandler(statisticsService)

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
	accountGroup.GET("/autoLogin", accountHandler.AutoLogin)
	accountGroup.POST("/autoLogin", accountHandler.AutoLogin)
	accountGroup.GET("/getUserCountInfo", middleware.WebAuth(accountService), accountHandler.GetUserCountInfo)
	accountGroup.POST("/getUserCountInfo", middleware.WebAuth(accountService), accountHandler.GetUserCountInfo)
	accountGroup.POST("/logout", middleware.WebAuth(accountService), accountHandler.Logout)

	uhomeGroup := group.Group("/uhome")
	uhomeGroup.GET("/getUserInfo", uhomeHandler.GetUserInfo)
	uhomeGroup.POST("/getUserInfo", uhomeHandler.GetUserInfo)
	uhomeGroup.POST("/updateUserInfo", middleware.WebAuth(accountService), uhomeHandler.UpdateUserInfo)
	uhomeGroup.POST("/focus", middleware.WebAuth(accountService), uhomeHandler.Focus)
	uhomeGroup.GET("/loadPeripheralCabinet", peripheralCabinetHandler.LoadPeripheralCabinet)
	uhomeGroup.POST("/loadPeripheralCabinet", peripheralCabinetHandler.LoadPeripheralCabinet)
	uhomeGroup.POST("/updatePeripheralCabinetVisible", middleware.WebAuth(accountService), peripheralCabinetHandler.UpdatePeripheralCabinetVisible)
	uhomeGroup.POST("/updatePeripheralCabinetItemVisible", middleware.WebAuth(accountService), peripheralCabinetHandler.UpdatePeripheralCabinetItemVisible)

	dynamicGroup := group.Group("/dynamic", middleware.WebAuth(accountService))
	dynamicGroup.POST("/loadCurrentUserInfo", dynamicHandler.LoadCurrentUserInfo)
	dynamicGroup.POST("/loadFollowUsers", dynamicHandler.LoadFollowUsers)
	dynamicGroup.POST("/loadFeed", dynamicHandler.LoadFeed)

	ucenterGroup := group.Group("/ucenter", middleware.WebAuth(accountService))
	ucenterGroup.POST("/postVideo", videoPostHandler.PostVideo)
	ucenterGroup.POST("/loadVideoList", videoPostHandler.LoadVideoList)
	ucenterGroup.GET("/getActualTimeStatisticsInfo", statisticsHandler.GetActualTimeStatisticsInfo)
	ucenterGroup.POST("/getActualTimeStatisticsInfo", statisticsHandler.GetActualTimeStatisticsInfo)
	ucenterGroup.GET("/getWeekStatisticsInfo", statisticsHandler.GetWeekStatisticsInfo)
	ucenterGroup.POST("/getWeekStatisticsInfo", statisticsHandler.GetWeekStatisticsInfo)

	fileGroup.POST("/preUploadVideo", middleware.WebAuth(accountService), videoUploadHandler.PreUploadVideo)
	fileGroup.POST("/uploadVideo", middleware.WebAuth(accountService), videoUploadHandler.UploadVideo)
	fileGroup.POST("/uploadImage", middleware.WebAuth(accountService), webFileHandler.UploadImage)
	fileGroup.GET("/videoResource/:fileId", webFileHandler.GetPublishedVideoResource)
	fileGroup.GET("/videoResource/:fileId/", webFileHandler.GetPublishedVideoResource)
	fileGroup.GET("/videoResource/:fileId/:resourceName", webFileHandler.GetPublishedVideoResourceSegment)
	fileGroup.GET("/downloadVideo/:fileId", webFileHandler.DownloadVideo)

	categoryGroup := group.Group("/category")
	categoryGroup.GET("/loadAllCategory", categoryHandler.LoadAllCategory)
	categoryGroup.POST("/loadAllCategory", categoryHandler.LoadAllCategory)

	videoGroup := group.Group("/video")
	videoGroup.GET("/loadRecommendVideo", videoHandler.LoadRecommendVideo)
	videoGroup.POST("/loadRecommendVideo", videoHandler.LoadRecommendVideo)
	videoGroup.GET("/loadVideo", videoHandler.LoadVideo)
	videoGroup.POST("/loadVideo", videoHandler.LoadVideo)
	videoGroup.GET("/search", videoHandler.Search)
	videoGroup.POST("/search", videoHandler.Search)
	videoGroup.GET("/getVideoInfo", videoHandler.GetVideoInfo)
	videoGroup.POST("/getVideoInfo", videoHandler.GetVideoInfo)
	videoGroup.GET("/loadVideoPList", videoHandler.LoadVideoPList)
	videoGroup.POST("/loadVideoPList", videoHandler.LoadVideoPList)

	sysSettingGroup := group.Group("/sysSetting")
	sysSettingGroup.GET("/getSetting", sysSettingHandler.GetSetting)
	sysSettingGroup.POST("/getSetting", sysSettingHandler.GetSetting)

	danmuGroup := interactGroup.Group("/danmu")
	danmuGroup.GET("/loadDanmu", danmuHandler.LoadDanmu)
	danmuGroup.POST("/loadDanmu", danmuHandler.LoadDanmu)
	danmuGroup.POST("/postDanmu", middleware.WebAuth(accountService), danmuHandler.PostDanmu)

	commentGroup := interactGroup.Group("/comment")
	commentGroup.GET("/loadComment", commentHandler.LoadComment)
	commentGroup.POST("/loadComment", commentHandler.LoadComment)
	commentGroup.Use(middleware.WebAuth(accountService))
	commentGroup.POST("/postComment", commentHandler.PostComment)

	userActionGroup := interactGroup.Group("/userAction", middleware.WebAuth(accountService))
	userActionGroup.POST("/doAction", userActionHandler.DoAction)

	onlineGroup := interactGroup.Group("/online")
	onlineGroup.GET("/reportVideoPlayOnline", onlineHandler.ReportVideoPlayOnline)
	onlineGroup.POST("/reportVideoPlayOnline", onlineHandler.ReportVideoPlayOnline)

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
	shopOrderGroup.POST("/cancel", shopOrderHandler.CancelOrder)
	shopOrderGroup.POST("/pay", shopOrderHandler.PayOrder)
}
