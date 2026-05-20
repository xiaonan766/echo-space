package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	shopSKUStockKeyPrefix        = "echo-space:shop:sku:stock:"
	shopOrderRequestKeyPrefix    = "echo-space:shop:order:request:"
	shopReservationKeyPrefix     = "echo-space:shop:stock:reservation:"
	shopReservationTimeoutKey    = "echo-space:shop:stock:reservation:timeout"
	shopSKUStockTTL              = 2 * time.Hour
	shopOrderRequestTTL          = 10 * time.Minute
	shopStockReservationTimeout  = 2 * time.Minute
	shopStockReservationKeepTTL  = 2 * time.Hour
	shopReleasedReservationTTL   = 10 * time.Minute
	shopLockedReservationKeepTTL = 10 * time.Minute
)

const (
	StockReservationStatusReserved     = "RESERVED"
	StockReservationStatusOrderCreated = "ORDER_CREATED"
	StockReservationStatusLocked       = "LOCKED"
	StockReservationStatusReleased     = "RELEASED"
)

const preDeductStockScript = `
local stockKey = KEYS[1]
local requestKey = KEYS[2]
local reservationKey = KEYS[3]
local timeoutKey = KEYS[4]
local orderNo = ARGV[1]
local userId = ARGV[2]
local productId = ARGV[3]
local skuId = ARGV[4]
local buyCount = tonumber(ARGV[5])
local requestTTL = tonumber(ARGV[6])
local reservationExpireAt = tonumber(ARGV[7])
local reservationTTL = tonumber(ARGV[8])

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

redis.call("HMSET", reservationKey,
	"orderNo", orderNo,
	"userId", userId,
	"productId", productId,
	"skuId", skuId,
	"buyCount", buyCount,
	"status", "RESERVED",
	"expireAt", reservationExpireAt
)
redis.call("EXPIRE", reservationKey, reservationTTL)
redis.call("ZADD", timeoutKey, reservationExpireAt, orderNo)
redis.call("SET", requestKey, orderNo, "EX", requestTTL)
redis.call("DECRBY", stockKey, buyCount)
return {1, orderNo}
`

const releaseReservationScript = `
local reservationKey = KEYS[1]
local stockKey = KEYS[2]
local timeoutKey = KEYS[3]
local orderNo = ARGV[1]
local now = ARGV[2]
local releasedTTL = tonumber(ARGV[3])
local stockTTL = tonumber(ARGV[4])

local status = redis.call("HGET", reservationKey, "status")
if not status then
	redis.call("ZREM", timeoutKey, orderNo)
	return {0, "missing"}
end
if status == "RELEASED" or status == "LOCKED" then
	redis.call("ZREM", timeoutKey, orderNo)
	return {2, status}
end

local buyCount = tonumber(redis.call("HGET", reservationKey, "buyCount") or "0")
if buyCount > 0 then
	redis.call("INCRBY", stockKey, buyCount)
	redis.call("EXPIRE", stockKey, stockTTL)
end
redis.call("HMSET", reservationKey, "status", "RELEASED", "releaseAt", now)
redis.call("ZREM", timeoutKey, orderNo)
redis.call("EXPIRE", reservationKey, releasedTTL)
return {1, "released"}
`

const markReservationReleasedScript = `
local reservationKey = KEYS[1]
local timeoutKey = KEYS[2]
local orderNo = ARGV[1]
local now = ARGV[2]
local releasedTTL = tonumber(ARGV[3])

local status = redis.call("HGET", reservationKey, "status")
if not status then
	redis.call("ZREM", timeoutKey, orderNo)
	return 0
end
redis.call("HMSET", reservationKey, "status", "RELEASED", "releaseAt", now)
redis.call("ZREM", timeoutKey, orderNo)
redis.call("EXPIRE", reservationKey, releasedTTL)
return 1
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

type StockReservation struct {
	OrderNo   string
	UserID    string
	ProductID uint64
	SkuID     uint64
	BuyCount  int
	Status    string
	ExpireAt  int64
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

func (s *ShopStockStore) PreDeductStock(ctx context.Context, userID string, requestID string, reservation StockReservation) (StockPreDeductResult, string, error) {
	if s == nil || s.redis == nil {
		return StockPreDeductInsufficient, "", errors.New("redis is not ready")
	}
	if reservation.ExpireAt <= 0 {
		reservation.ExpireAt = time.Now().Add(shopStockReservationTimeout).Unix()
	}

	result, err := s.redis.Eval(ctx, preDeductStockScript, []string{
		skuStockKey(reservation.SkuID),
		orderRequestKey(userID, requestID),
		reservationKey(reservation.OrderNo),
		shopReservationTimeoutKey,
	},
		reservation.OrderNo,
		reservation.UserID,
		strconv.FormatUint(reservation.ProductID, 10),
		strconv.FormatUint(reservation.SkuID, 10),
		reservation.BuyCount,
		int(shopOrderRequestTTL.Seconds()),
		reservation.ExpireAt,
		int(shopStockReservationKeepTTL.Seconds()),
	).Result()
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

func (s *ShopStockStore) MarkReservationOrderCreated(ctx context.Context, orderNo string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil
	}
	exists, err := s.redis.Exists(ctx, reservationKey(orderNo)).Result()
	if err != nil || exists == 0 {
		return err
	}
	return s.redis.HSet(ctx, reservationKey(orderNo), "status", StockReservationStatusOrderCreated).Err()
}

func (s *ShopStockStore) MarkReservationLocked(ctx context.Context, orderNo string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil
	}
	key := reservationKey(orderNo)
	pipe := s.redis.TxPipeline()
	pipe.HSet(ctx, key, "status", StockReservationStatusLocked)
	pipe.ZRem(ctx, shopReservationTimeoutKey, orderNo)
	pipe.Expire(ctx, key, shopLockedReservationKeepTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *ShopStockStore) MarkReservationReleased(ctx context.Context, orderNo string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil
	}
	return s.redis.Eval(ctx, markReservationReleasedScript, []string{
		reservationKey(orderNo),
		shopReservationTimeoutKey,
	}, orderNo, time.Now().Unix(), int(shopReleasedReservationTTL.Seconds())).Err()
}

func (s *ShopStockStore) ReleaseReservation(ctx context.Context, orderNo string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil
	}
	reservation, ok, err := s.GetReservation(ctx, orderNo)
	if err != nil || !ok {
		return err
	}
	return s.redis.Eval(ctx, releaseReservationScript, []string{
		reservationKey(orderNo),
		skuStockKey(reservation.SkuID),
		shopReservationTimeoutKey,
	}, orderNo, time.Now().Unix(), int(shopReleasedReservationTTL.Seconds()), int(shopSKUStockTTL.Seconds())).Err()
}

func (s *ShopStockStore) ListExpiredReservations(ctx context.Context, nowUnix int64, limit int64) ([]string, error) {
	if s == nil || s.redis == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	return s.redis.ZRangeByScore(ctx, shopReservationTimeoutKey, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(nowUnix, 10),
		Offset: 0,
		Count:  limit,
	}).Result()
}

func (s *ShopStockStore) GetReservation(ctx context.Context, orderNo string) (*StockReservation, bool, error) {
	if s == nil || s.redis == nil {
		return nil, false, nil
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, false, nil
	}
	values, err := s.redis.HGetAll(ctx, reservationKey(orderNo)).Result()
	if err != nil {
		return nil, false, err
	}
	if len(values) == 0 {
		return nil, false, nil
	}
	reservation := &StockReservation{
		OrderNo: values["orderNo"],
		UserID:  values["userId"],
		Status:  values["status"],
	}
	reservation.ProductID, _ = strconv.ParseUint(values["productId"], 10, 64)
	reservation.SkuID, _ = strconv.ParseUint(values["skuId"], 10, 64)
	reservation.BuyCount, _ = strconv.Atoi(values["buyCount"])
	reservation.ExpireAt, _ = strconv.ParseInt(values["expireAt"], 10, 64)
	return reservation, true, nil
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

func reservationKey(orderNo string) string {
	return shopReservationKeyPrefix + orderNo
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
