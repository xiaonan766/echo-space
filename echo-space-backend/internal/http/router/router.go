package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	filehandler "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/handler"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/mq"
)

type Dependencies struct {
	Config                  config.Config
	Redis                   *redis.Client
	Cache                   *cache.HybridCache
	DB                      *gorm.DB
	StockLockPublisher      *mq.ShopStockLockPublisher
	VideoTranscodePublisher *mq.VideoTranscodePublisher
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

	fileHandler := filehandler.NewFileHandler(deps.Config.File)
	fileGroup := engine.Group("/file")
	fileGroup.GET("/getResource", fileHandler.GetResource)

	registerAdminRoutes(engine.Group("/admin"), deps)
	registerWebRoutes(engine.Group("/web"), fileGroup, deps)

	return engine
}
