package web

import (
	"context"
	"log"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	shopStockPrewarmInterval  = time.Minute
	shopStockPrewarmWindow    = 5 * time.Minute
	shopStockPrewarmBatchSize = 500
)

type ShopStockPrewarmService struct {
	shopRepository *repository.ShopRepository
	stockStore     *cache.ShopStockStore
}

func NewShopStockPrewarmService(shopRepository *repository.ShopRepository, stockStore *cache.ShopStockStore) *ShopStockPrewarmService {
	return &ShopStockPrewarmService{
		shopRepository: shopRepository,
		stockStore:     stockStore,
	}
}

func (s *ShopStockPrewarmService) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *ShopStockPrewarmService) run(ctx context.Context) {
	s.PrewarmStartingSoon(ctx)

	ticker := time.NewTicker(shopStockPrewarmInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.PrewarmStartingSoon(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *ShopStockPrewarmService) PrewarmStartingSoon(ctx context.Context) {
	if s == nil || s.shopRepository == nil || s.stockStore == nil {
		return
	}

	now := time.Now()
	list, err := s.shopRepository.ListPeripheralSKUStockStartingSoon(ctx, now, now.Add(shopStockPrewarmWindow), shopStockPrewarmBatchSize)
	if err != nil {
		log.Printf("list starting soon peripheral sku stock failed: %v", err)
		return
	}

	for _, item := range list {
		if err := s.stockStore.PrewarmSKUStock(ctx, item.SkuID, item.AvailableStock); err != nil {
			log.Printf("prewarm starting soon peripheral sku stock failed: productID=%d skuID=%d err=%v", item.ProductID, item.SkuID, err)
		}
	}
}
