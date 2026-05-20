package web

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type ShopCacheRecoveryHandler struct {
	shopRepository *repository.ShopRepository
	recommendStore *cache.ShopRecommendStore
}

func NewShopCacheRecoveryHandler(shopRepository *repository.ShopRepository, recommendStore *cache.ShopRecommendStore) *ShopCacheRecoveryHandler {
	return &ShopCacheRecoveryHandler{
		shopRepository: shopRepository,
		recommendStore: recommendStore,
	}
}

func (h *ShopCacheRecoveryHandler) HandleShopCacheRecoveryTask(ctx context.Context, task cache.ShopCacheRecoveryTask) error {
	if h == nil || h.shopRepository == nil || h.recommendStore == nil {
		return nil
	}

	if _, ok := task.RemainingTTL(); !ok {
		return nil
	}

	switch task.Type {
	case cache.ShopCacheRecoveryTaskRebuildPeripheralDetail:
		return h.rebuildPeripheralDetail(ctx, task.ProductID, task.Hot)
	case cache.ShopCacheRecoveryTaskSetHotPeripheralMarker:
		return h.restoreHotPeripheralMarker(ctx, task)
	case cache.ShopCacheRecoveryTaskRebuildHotPeripheral:
		return h.rebuildHotPeripheralRecommend(ctx)
	case cache.ShopCacheRecoveryTaskDeleteKey:
		return h.deleteShopCacheKey(ctx, task.Key)
	default:
		return nil
	}
}

func (h *ShopCacheRecoveryHandler) rebuildPeripheralDetail(ctx context.Context, productID uint64, hot bool) error {
	if productID == 0 {
		return nil
	}

	item, err := h.shopRepository.FindWebPeripheralDetail(ctx, productID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return h.recommendStore.DeletePeripheralDetailFromRedis(ctx, productID)
	}
	if err != nil {
		return err
	}

	fillWebShopItem(item)
	return h.recommendStore.SavePeripheralDetailToRedis(ctx, productID, item, hot)
}

func (h *ShopCacheRecoveryHandler) restoreHotPeripheralMarker(ctx context.Context, task cache.ShopCacheRecoveryTask) error {
	ttl, ok := task.RemainingTTL()
	if !ok {
		return nil
	}
	return h.recommendStore.RestoreHotPeripheralMarkerToRedis(ctx, task.Key, []byte(task.Value), ttl)
}

func (h *ShopCacheRecoveryHandler) rebuildHotPeripheralRecommend(ctx context.Context) error {
	list, err := h.shopRepository.ListRecommendedPeripheralForWeb(ctx, defaultRecommendSize)
	if err != nil {
		return err
	}
	fillWebShopItems(list)
	return h.recommendStore.SaveHotPeripheralToRedis(ctx, list)
}

func (h *ShopCacheRecoveryHandler) deleteShopCacheKey(ctx context.Context, key string) error {
	return h.recommendStore.DeleteShopCacheKeyFromRedis(ctx, key)
}
