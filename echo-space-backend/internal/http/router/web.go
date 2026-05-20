package router

import (
	"github.com/gin-gonic/gin"

	webhandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler/web"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

func registerWebRoutes(group *gin.RouterGroup, deps Dependencies) {
	userRepository := repository.NewUserRepository(deps.DB)
	accountService := webservice.NewAccountService(deps.Cache, userRepository)
	accountHandler := webhandler.NewAccountHandler(accountService)

	categoryRepository := repository.NewCategoryRepository(deps.DB)
	categoryService := webservice.NewCategoryService(categoryRepository)
	categoryHandler := webhandler.NewCategoryHandler(categoryService)

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

	categoryGroup := group.Group("/category")
	categoryGroup.GET("/loadAllCategory", categoryHandler.LoadAllCategory)
	categoryGroup.POST("/loadAllCategory", categoryHandler.LoadAllCategory)

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
