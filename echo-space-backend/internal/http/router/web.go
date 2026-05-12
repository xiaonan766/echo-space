package router

import (
	"github.com/gin-gonic/gin"

	webhandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler/web"
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

	accountGroup := group.Group("/account")
	accountGroup.GET("/checkCode", accountHandler.CheckCode)
	accountGroup.POST("/checkCode", accountHandler.CheckCode)
	accountGroup.POST("/register", accountHandler.Register)

	categoryGroup := group.Group("/category")
	categoryGroup.GET("/loadAllCategory", categoryHandler.LoadAllCategory)
	categoryGroup.POST("/loadAllCategory", categoryHandler.LoadAllCategory)
}
