package router

import (
	"github.com/gin-gonic/gin"

	webhandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler/web"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
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
	shopService := webservice.NewShopService(shopRepository)
	shopHandler := webhandler.NewShopHandler(shopService)

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
}
