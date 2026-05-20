package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	shopSKUStockKeyPrefix     = "echo-space:shop:sku:stock:"
	shopOrderRequestKeyPrefix = "echo-space:shop:order:request:"
	shopSKUStockTTL           = 2 * time.Hour
	shopOrderRequestTTL       = 10 * time.Minute
)

const preDeductStockScript = `
local stockKey = KEYS[1]
local requestKey = KEYS[2]
local orderNo = ARGV[1]
local buyCount = tonumber(ARGV[2])
local requestTTL = tonumber(ARGV[3])

local existingOrderNo = redis.call("GET", requestKey)
if existingOrderNo then
	return {2, existingOrderNo}
end

local stock = redis.call("GET", stockKey)
if not stock then
	return {-1, ""}
end

if tonumber(stock) < buyCount then
	return {0, ""}
end

redis.call("DECRBY", stockKey, buyCount)
redis.call("SET", requestKey, orderNo, "EX", requestTTL)
return {1, orderNo}
`

type StockPreDeductResult int

const (
	StockPreDeductMissing      StockPreDeductResult = -1
	StockPreDeductInsufficient StockPreDeductResult = 0
	StockPreDeductSuccess      StockPreDeductResult = 1
	StockPreDeductRepeated     StockPreDeductResult = 2
)

type ShopStockStore struct {
	redis *redis.Client
}

func NewShopStockStore(redisClient *redis.Client) *ShopStockStore {
	return &ShopStockStore{
		redis: redisClient,
	}
}

func (s *ShopStockStore) InitSKUStockIfAbsent(ctx context.Context, skuID uint64, availableStock int) error {
	if s == nil || s.redis == nil {
		return errors.New("redis is not ready")
	}
	if skuID == 0 || availableStock < 0 {
		return nil
	}
	return s.redis.SetNX(ctx, skuStockKey(skuID), availableStock, shopSKUStockTTL).Err()
}

func (s *ShopStockStore) ResetSKUStock(ctx context.Context, skuID uint64, availableStock int) error {
	if s == nil || s.redis == nil {
		return errors.New("redis is not ready")
	}
	if skuID == 0 {
		return nil
	}
	if availableStock < 0 {
		availableStock = 0
	}
	return s.redis.Set(ctx, skuStockKey(skuID), availableStock, shopSKUStockTTL).Err()
}

func (s *ShopStockStore) PreDeductStock(ctx context.Context, userID string, requestID string, skuID uint64, orderNo string, buyCount int) (StockPreDeductResult, string, error) {
	if s == nil || s.redis == nil {
		return StockPreDeductInsufficient, "", errors.New("redis is not ready")
	}

	result, err := s.redis.Eval(ctx, preDeductStockScript, []string{
		skuStockKey(skuID),
		orderRequestKey(userID, requestID),
	}, orderNo, buyCount, int(shopOrderRequestTTL.Seconds())).Result()
	if err != nil {
		return StockPreDeductInsufficient, "", err
	}

	values, ok := result.([]any)
	if !ok || len(values) < 2 {
		return StockPreDeductInsufficient, "", fmt.Errorf("unexpected redis stock result: %v", result)
	}

	code, err := parseRedisInt(values[0])
	if err != nil {
		return StockPreDeductInsufficient, "", err
	}
	existingOrderNo := fmt.Sprint(values[1])
	return StockPreDeductResult(code), existingOrderNo, nil
}

func (s *ShopStockStore) CompensateStock(ctx context.Context, skuID uint64, buyCount int) error {
	if s == nil || s.redis == nil || skuID == 0 || buyCount <= 0 {
		return nil
	}
	return s.redis.IncrBy(ctx, skuStockKey(skuID), int64(buyCount)).Err()
}

func (s *ShopStockStore) DeleteRequest(ctx context.Context, userID string, requestID string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, orderRequestKey(userID, requestID)).Err()
}

func skuStockKey(skuID uint64) string {
	return fmt.Sprintf("%s%d", shopSKUStockKeyPrefix, skuID)
}

func orderRequestKey(userID string, requestID string) string {
	return shopOrderRequestKeyPrefix + userID + ":" + requestID
}

func parseRedisInt(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case string:
		return strconv.Atoi(typed)
	case []byte:
		return strconv.Atoi(string(typed))
	default:
		return 0, fmt.Errorf("unexpected redis integer type %T", value)
	}
}
