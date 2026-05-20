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
	shopStockLockConsumer     *mq.ShopStockLockConsumer
	backgroundCancel          context.CancelFunc
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

	rabbitClient := setupRabbit(ctx, cfg)
	shopCacheRecoveryConsumer := setupShopCacheRecovery(ctx, cfg, hybridCache, redisClient, mysqlDB, rabbitClient)
	stockLockPublisher, shopStockLockConsumer := setupShopStockLock(ctx, cfg, redisClient, mysqlDB, rabbitClient)
	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())
	setupShopOrderRecovery(backgroundCtx, redisClient, mysqlDB, stockLockPublisher)

	router := approuter.New(approuter.Dependencies{
		Config:             cfg,
		Redis:              redisClient,
		Cache:              hybridCache,
		DB:                 mysqlDB,
		StockLockPublisher: stockLockPublisher,
	})

	return &App{
		cfg:                       cfg,
		redis:                     redisClient,
		cache:                     hybridCache,
		db:                        mysqlDB,
		rabbit:                    rabbitClient,
		shopCacheRecoveryConsumer: shopCacheRecoveryConsumer,
		shopStockLockConsumer:     shopStockLockConsumer,
		backgroundCancel:          backgroundCancel,
		router:                    router,
	}, nil
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) Close() {
	if a.backgroundCancel != nil {
		a.backgroundCancel()
	}
	if a.shopStockLockConsumer != nil {
		a.shopStockLockConsumer.Close()
	}
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

func setupRabbit(ctx context.Context, cfg config.Config) *mq.RabbitClient {
	rabbitClient, err := mq.NewRabbitClient(ctx, cfg.RabbitMQ)
	if err != nil {
		log.Printf("rabbitmq unavailable, async shop tasks will be disabled: %v", err)
		return nil
	}
	return rabbitClient
}

func setupShopCacheRecovery(ctx context.Context, cfg config.Config, hybridCache *cache.HybridCache, redisClient *redis.Client, mysqlDB *gorm.DB, rabbitClient *mq.RabbitClient) *mq.ShopCacheRecoveryConsumer {
	if rabbitClient == nil {
		log.Printf("cache recovery will use direct redis write because rabbitmq is unavailable")
		return nil
	}

	shopRepository := repository.NewShopRepository(mysqlDB)
	shopRecommendStore := cache.NewShopRecommendStore(hybridCache, redisClient)
	recoveryHandler := webservice.NewShopCacheRecoveryHandler(shopRepository, shopRecommendStore)
	consumer := mq.NewShopCacheRecoveryConsumer(rabbitClient, cfg.RabbitMQ.CacheRecoveryQueue, cfg.RabbitMQ.PrefetchCount, recoveryHandler)
	if err := consumer.Start(ctx); err != nil {
		log.Printf("start shop cache recovery consumer failed: %v", err)
		return nil
	}

	publisher := mq.NewShopCacheRecoveryPublisher(rabbitClient, cfg.RabbitMQ.CacheRecoveryQueue)
	hybridCache.SetRecoveryHandler(cache.NewShopCacheRecoveryHandler(publisher))

	log.Printf("shop cache recovery consumer started, queue=%s", cfg.RabbitMQ.CacheRecoveryQueue)
	return consumer
}

func setupShopStockLock(ctx context.Context, cfg config.Config, redisClient *redis.Client, mysqlDB *gorm.DB, rabbitClient *mq.RabbitClient) (*mq.ShopStockLockPublisher, *mq.ShopStockLockConsumer) {
	if rabbitClient == nil {
		log.Printf("shop stock lock is disabled because rabbitmq is unavailable")
		return nil, nil
	}

	shopRepository := repository.NewShopRepository(mysqlDB)
	orderRepository := repository.NewShopOrderRepository(mysqlDB)
	stockStore := cache.NewShopStockStore(redisClient)
	orderService := webservice.NewShopOrderService(shopRepository, orderRepository, stockStore, nil)
	consumer := mq.NewShopStockLockConsumer(rabbitClient, cfg.RabbitMQ.StockLockQueue, cfg.RabbitMQ.PrefetchCount, orderService)
	if err := consumer.Start(ctx); err != nil {
		log.Printf("start shop stock lock consumer failed: %v", err)
		return nil, nil
	}

	publisher := mq.NewShopStockLockPublisher(rabbitClient, cfg.RabbitMQ.StockLockQueue)
	log.Printf("shop stock lock consumer started, queue=%s", cfg.RabbitMQ.StockLockQueue)
	return publisher, consumer
}

func setupShopOrderRecovery(ctx context.Context, redisClient *redis.Client, mysqlDB *gorm.DB, stockLockPublisher *mq.ShopStockLockPublisher) {
	shopRepository := repository.NewShopRepository(mysqlDB)
	orderRepository := repository.NewShopOrderRepository(mysqlDB)
	stockStore := cache.NewShopStockStore(redisClient)
	orderService := webservice.NewShopOrderService(shopRepository, orderRepository, stockStore, stockLockPublisher)
	orderService.StartRecoveryTasks(ctx)
	log.Printf("shop order recovery tasks started")
}
