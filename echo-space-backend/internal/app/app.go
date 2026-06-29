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
	searchinfra "github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/search"
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
	videoTranscodeConsumer    *mq.VideoTranscodeConsumer
	dynamicFeedConsumer       *mq.DynamicFeedConsumer
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

	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())
	rabbitClient := setupRabbit(ctx, cfg)
	shopCacheRecoveryConsumer := setupShopCacheRecovery(backgroundCtx, cfg, hybridCache, redisClient, mysqlDB, rabbitClient)
	stockLockPublisher, shopStockLockConsumer := setupShopStockLock(backgroundCtx, cfg, redisClient, mysqlDB, rabbitClient)
	videoTranscodePublisher, videoTranscodeConsumer := setupVideoTranscode(backgroundCtx, cfg, redisClient, mysqlDB, rabbitClient)
	dynamicFeedConsumer := setupDynamicFeed(backgroundCtx, cfg, redisClient, mysqlDB, rabbitClient)
	setupShopOrderRecovery(backgroundCtx, redisClient, mysqlDB, stockLockPublisher)
	setupShopStockPrewarm(backgroundCtx, redisClient, mysqlDB)
	videoSearch := setupVideoSearch(ctx, backgroundCtx, cfg, mysqlDB)
	searchKeywordStore := cache.NewSearchKeywordStore(redisClient)

	router := approuter.New(approuter.Dependencies{
		Config:                  cfg,
		Redis:                   redisClient,
		Cache:                   hybridCache,
		DB:                      mysqlDB,
		VideoSearch:             videoSearch,
		SearchKeywordStore:      searchKeywordStore,
		StockLockPublisher:      stockLockPublisher,
		VideoTranscodePublisher: videoTranscodePublisher,
	})

	return &App{
		cfg:                       cfg,
		redis:                     redisClient,
		cache:                     hybridCache,
		db:                        mysqlDB,
		rabbit:                    rabbitClient,
		shopCacheRecoveryConsumer: shopCacheRecoveryConsumer,
		shopStockLockConsumer:     shopStockLockConsumer,
		videoTranscodeConsumer:    videoTranscodeConsumer,
		dynamicFeedConsumer:       dynamicFeedConsumer,
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
	if a.videoTranscodeConsumer != nil {
		a.videoTranscodeConsumer.Close()
	}
	if a.dynamicFeedConsumer != nil {
		a.dynamicFeedConsumer.Close()
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
		log.Printf("rabbitmq configuration is invalid, async tasks will be disabled: %v", err)
		return nil
	}
	return rabbitClient
}

func setupVideoTranscode(ctx context.Context, cfg config.Config, redisClient *redis.Client, mysqlDB *gorm.DB, rabbitClient *mq.RabbitClient) (*mq.VideoTranscodePublisher, *mq.VideoTranscodeConsumer) {
	if rabbitClient == nil {
		log.Printf("video transcode publisher is disabled because rabbitmq is not configured")
		return nil, nil
	}

	repo := repository.NewVideoPostRepository(mysqlDB)
	uploadStore := cache.NewUploadingFileStore(redisClient)
	publisher := mq.NewVideoTranscodePublisher(rabbitClient, cfg.RabbitMQ.VideoTranscodeQueue)
	service := webservice.NewVideoTranscodeService(repo, uploadStore, publisher, cfg.File.ResourceRoot)
	consumer := mq.NewVideoTranscodeConsumer(rabbitClient, cfg.RabbitMQ.VideoTranscodeQueue, cfg.RabbitMQ.VideoTranscodePrefetch, service)
	if err := consumer.Start(ctx); err != nil {
		log.Printf("start video transcode consumer failed: %v", err)
		return publisher, nil
	}
	service.StartOutboxPublisher(ctx)
	log.Printf("video transcode consumer started, queue=%s", cfg.RabbitMQ.VideoTranscodeQueue)
	return publisher, consumer
}

func setupDynamicFeed(ctx context.Context, cfg config.Config, redisClient *redis.Client, mysqlDB *gorm.DB, rabbitClient *mq.RabbitClient) *mq.DynamicFeedConsumer {
	if rabbitClient == nil {
		log.Printf("dynamic feed publisher is disabled because rabbitmq is not configured")
		return nil
	}

	repo := repository.NewDynamicRepository(mysqlDB)
	activeStore := cache.NewDynamicActiveStore(redisClient)
	publisher := mq.NewDynamicFeedPublisher(rabbitClient, cfg.RabbitMQ.DynamicFeedQueue)
	service := webservice.NewDynamicFeedService(repo, activeStore, publisher, activeStore)
	consumer := mq.NewDynamicFeedConsumer(rabbitClient, cfg.RabbitMQ.DynamicFeedQueue, cfg.RabbitMQ.PrefetchCount, service)
	if err := consumer.Start(ctx); err != nil {
		log.Printf("start dynamic feed consumer failed: %v", err)
		return nil
	}
	service.StartOutboxPublisher(ctx)
	log.Printf("dynamic feed consumer started, queue=%s", cfg.RabbitMQ.DynamicFeedQueue)
	return consumer
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

func setupShopStockPrewarm(ctx context.Context, redisClient *redis.Client, mysqlDB *gorm.DB) {
	shopRepository := repository.NewShopRepository(mysqlDB)
	stockStore := cache.NewShopStockStore(redisClient)
	prewarmService := webservice.NewShopStockPrewarmService(shopRepository, stockStore)
	prewarmService.Start(ctx)
	log.Printf("shop stock prewarm task started")
}

func setupVideoSearch(startupCtx context.Context, backgroundCtx context.Context, cfg config.Config, mysqlDB *gorm.DB) *searchinfra.VideoIndex {
	videoSearch, err := searchinfra.NewVideoIndex(cfg.Elasticsearch)
	if err != nil {
		log.Printf("elasticsearch video search is disabled: %v", err)
		return nil
	}
	if err := videoSearch.EnsureVideoIndex(startupCtx); err != nil {
		log.Printf("ensure elasticsearch video index failed: %v", err)
		return videoSearch
	}

	videoRepository := repository.NewVideoRepository(mysqlDB)
	go backfillVideoSearchDocuments(backgroundCtx, videoSearch, videoRepository)
	log.Printf("elasticsearch video search initialized, index=%s", cfg.Elasticsearch.IndexVideoName)
	return videoSearch
}

func backfillVideoSearchDocuments(ctx context.Context, videoSearch *searchinfra.VideoIndex, videoRepository *repository.VideoRepository) {
	const pageSize = 200

	for offset := 0; ; offset += pageSize {
		select {
		case <-ctx.Done():
			return
		default:
		}

		list, err := videoRepository.ListVideoSearchDocuments(ctx, offset, pageSize)
		if err != nil {
			log.Printf("backfill video search documents failed: %v", err)
			return
		}
		if len(list) == 0 {
			return
		}

		for _, document := range list {
			if err := videoSearch.IndexVideo(ctx, document); err != nil {
				log.Printf("index video search document failed: videoID=%s err=%v", document.VideoID, err)
			}
		}
	}
}
