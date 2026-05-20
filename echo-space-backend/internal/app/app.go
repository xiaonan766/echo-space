package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	approuter "github.com/xiaonan766/echo-space/echo-space-backend/internal/http/router"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/database"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/mq"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type App struct {
	cfg                       config.Config
	redis                     *redis.Client
	cache                     *cache.HybridCache
	db                        *gorm.DB
	rabbit                    *mq.RabbitClient
	shopCacheRecoveryConsumer *mq.ShopCacheRecoveryConsumer
	router                    http.Handler
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}
	hybridCache := cache.NewHybridCache(redisClient)

	mysqlDB, err := database.NewMySQLClient(ctx, cfg.MySQL)
	if err != nil {
		hybridCache.Close()
		_ = redisClient.Close()
		return nil, err
	}

	if cfg.MySQL.AutoMigrate {
		if err := mysqlDB.AutoMigrate(&domain.CategoryInfo{}); err != nil {
			hybridCache.Close()
			_ = redisClient.Close()
			closeDB(mysqlDB)
			return nil, fmt.Errorf("auto migrate mysql: %w", err)
		}
	}

	rabbitClient, shopCacheRecoveryConsumer := setupShopCacheRecovery(ctx, cfg, hybridCache, redisClient, mysqlDB)

	router := approuter.New(approuter.Dependencies{
		Config: cfg,
		Redis:  redisClient,
		Cache:  hybridCache,
		DB:     mysqlDB,
	})

	return &App{
		cfg:                       cfg,
		redis:                     redisClient,
		cache:                     hybridCache,
		db:                        mysqlDB,
		rabbit:                    rabbitClient,
		shopCacheRecoveryConsumer: shopCacheRecoveryConsumer,
		router:                    router,
	}, nil
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) Close() {
	if a.shopCacheRecoveryConsumer != nil {
		a.shopCacheRecoveryConsumer.Close()
	}
	if a.rabbit != nil {
		a.rabbit.Close()
	}
	if a.cache != nil {
		a.cache.Close()
	}
	if a.redis != nil {
		_ = a.redis.Close()
	}
	closeDB(a.db)
}

func closeDB(db *gorm.DB) {
	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

func setupShopCacheRecovery(ctx context.Context, cfg config.Config, hybridCache *cache.HybridCache, redisClient *redis.Client, mysqlDB *gorm.DB) (*mq.RabbitClient, *mq.ShopCacheRecoveryConsumer) {
	rabbitClient, err := mq.NewRabbitClient(ctx, cfg.RabbitMQ)
	if err != nil {
		log.Printf("rabbitmq unavailable, cache recovery will use direct redis write: %v", err)
		return nil, nil
	}

	shopRepository := repository.NewShopRepository(mysqlDB)
	shopRecommendStore := cache.NewShopRecommendStore(hybridCache, redisClient)
	recoveryHandler := webservice.NewShopCacheRecoveryHandler(shopRepository, shopRecommendStore)
	consumer := mq.NewShopCacheRecoveryConsumer(rabbitClient, cfg.RabbitMQ.CacheRecoveryQueue, cfg.RabbitMQ.PrefetchCount, recoveryHandler)
	if err := consumer.Start(ctx); err != nil {
		log.Printf("start shop cache recovery consumer failed: %v", err)
		return rabbitClient, nil
	}

	publisher := mq.NewShopCacheRecoveryPublisher(rabbitClient, cfg.RabbitMQ.CacheRecoveryQueue)
	hybridCache.SetRecoveryHandler(cache.NewShopCacheRecoveryHandler(publisher))

	log.Printf("shop cache recovery consumer started, queue=%s", cfg.RabbitMQ.CacheRecoveryQueue)
	return rabbitClient, consumer
}
