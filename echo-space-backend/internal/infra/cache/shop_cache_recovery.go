package cache

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type ShopCacheRecoveryTaskType string

const (
	ShopCacheRecoveryTaskRebuildPeripheralDetail ShopCacheRecoveryTaskType = "rebuild_peripheral_detail"
	ShopCacheRecoveryTaskSetHotPeripheralMarker  ShopCacheRecoveryTaskType = "set_hot_peripheral_marker"
	ShopCacheRecoveryTaskRebuildHotPeripheral    ShopCacheRecoveryTaskType = "rebuild_hot_peripheral"
	ShopCacheRecoveryTaskDeleteKey               ShopCacheRecoveryTaskType = "delete_key"
	shopCacheRecoveryTaskIgnore                  ShopCacheRecoveryTaskType = "ignore"
)

type ShopCacheRecoveryTask struct {
	Type          ShopCacheRecoveryTaskType `json:"type"`
	ProductID     uint64                    `json:"productId,omitempty"`
	Hot           bool                      `json:"hot,omitempty"`
	Key           string                    `json:"key,omitempty"`
	Value         string                    `json:"value,omitempty"`
	ExpireAtMilli int64                     `json:"expireAtMilli,omitempty"`
}

type ShopCacheRecoveryPublisher interface {
	PublishShopCacheRecoveryTask(ctx context.Context, task ShopCacheRecoveryTask) error
}

type shopCacheRecoveryHandler struct {
	publisher ShopCacheRecoveryPublisher
}

func NewShopCacheRecoveryHandler(publisher ShopCacheRecoveryPublisher) RecoveryHandler {
	return &shopCacheRecoveryHandler{
		publisher: publisher,
	}
}

func (h *shopCacheRecoveryHandler) HandleDirtyWrite(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	if h == nil || h.publisher == nil {
		return false, nil
	}

	task, ok := BuildShopCacheRecoveryTask(key, value, ttl)
	if !ok {
		return false, nil
	}
	if task.Type == shopCacheRecoveryTaskIgnore {
		return true, nil
	}
	return true, h.publisher.PublishShopCacheRecoveryTask(ctx, task)
}

func (h *shopCacheRecoveryHandler) HandlePendingDelete(ctx context.Context, key string) (bool, error) {
	if h == nil || h.publisher == nil {
		return false, nil
	}

	task, ok := BuildShopCacheDeleteTask(key)
	if !ok {
		return false, nil
	}
	if task.Type == shopCacheRecoveryTaskIgnore {
		return true, nil
	}
	return true, h.publisher.PublishShopCacheRecoveryTask(ctx, task)
}

func BuildShopCacheRecoveryTask(key string, value []byte, ttl time.Duration) (ShopCacheRecoveryTask, bool) {
	expireAtMilli := int64(0)
	if ttl > 0 {
		expireAtMilli = time.Now().Add(ttl).UnixMilli()
	}

	switch {
	case key == hotPeripheralRecommendKey:
		return ShopCacheRecoveryTask{
			Type:          ShopCacheRecoveryTaskRebuildHotPeripheral,
			ExpireAtMilli: expireAtMilli,
		}, true
	case strings.HasPrefix(key, hotPeripheralDetailKeyPrefix):
		productID, ok := parseShopCacheProductID(key, hotPeripheralDetailKeyPrefix)
		if !ok {
			return ShopCacheRecoveryTask{}, false
		}
		return ShopCacheRecoveryTask{
			Type:          ShopCacheRecoveryTaskRebuildPeripheralDetail,
			ProductID:     productID,
			Hot:           true,
			ExpireAtMilli: expireAtMilli,
		}, true
	case strings.HasPrefix(key, peripheralDetailKeyPrefix):
		productID, ok := parseShopCacheProductID(key, peripheralDetailKeyPrefix)
		if !ok {
			return ShopCacheRecoveryTask{}, false
		}
		return ShopCacheRecoveryTask{
			Type:          ShopCacheRecoveryTaskRebuildPeripheralDetail,
			ProductID:     productID,
			Hot:           false,
			ExpireAtMilli: expireAtMilli,
		}, true
	case strings.HasPrefix(key, hotPeripheralMarkerKeyPrefix):
		productID, ok := parseShopCacheProductID(key, hotPeripheralMarkerKeyPrefix)
		if !ok {
			return ShopCacheRecoveryTask{}, false
		}
		return ShopCacheRecoveryTask{
			Type:          ShopCacheRecoveryTaskSetHotPeripheralMarker,
			ProductID:     productID,
			Key:           key,
			Value:         string(value),
			ExpireAtMilli: expireAtMilli,
		}, true
	case strings.HasPrefix(key, hotPeripheralScoreKeyPrefix):
		return ShopCacheRecoveryTask{Type: shopCacheRecoveryTaskIgnore}, true
	default:
		return ShopCacheRecoveryTask{}, false
	}
}

func BuildShopCacheDeleteTask(key string) (ShopCacheRecoveryTask, bool) {
	switch {
	case key == hotPeripheralRecommendKey,
		strings.HasPrefix(key, hotPeripheralDetailKeyPrefix),
		strings.HasPrefix(key, peripheralDetailKeyPrefix),
		strings.HasPrefix(key, hotPeripheralMarkerKeyPrefix):
		return ShopCacheRecoveryTask{
			Type: ShopCacheRecoveryTaskDeleteKey,
			Key:  key,
		}, true
	case strings.HasPrefix(key, hotPeripheralScoreKeyPrefix):
		return ShopCacheRecoveryTask{Type: shopCacheRecoveryTaskIgnore}, true
	default:
		return ShopCacheRecoveryTask{}, false
	}
}

func (t ShopCacheRecoveryTask) RemainingTTL() (time.Duration, bool) {
	if t.ExpireAtMilli == 0 {
		return 0, true
	}

	remaining := time.Until(time.UnixMilli(t.ExpireAtMilli))
	if remaining <= 0 {
		return 0, false
	}
	return remaining, true
}

func parseShopCacheProductID(key string, prefix string) (uint64, bool) {
	rawID := strings.TrimPrefix(key, prefix)
	productID, err := strconv.ParseUint(rawID, 10, 64)
	return productID, err == nil && productID > 0
}
