package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
)

type Dependencies struct {
	Config config.Config
	Redis  *redis.Client
}

func New(deps Dependencies) *gin.Engine {
	gin.SetMode(deps.Config.Server.Mode)

	engine := gin.New()
	engine.Use(gin.Logger(), middleware.Recovery())
	engine.NoRoute(response.NotFound)

	engine.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"service": "echo-space-backend",
		})
	})

	registerAdminRoutes(engine.Group("/admin"), deps)

	return engine
}
