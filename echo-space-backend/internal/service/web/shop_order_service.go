package web

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/mq"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultOrderPageNo        = 1
	defaultOrderPageSize      = 10
	maxOrderPageSize          = 50
	defaultOrderExpireMinutes = 15
	defaultOrderSkuName       = "默认规格"
)

type ShopStockLockPublisher interface {
	PublishShopStockLockMessage(ctx context.Context, message mq.ShopStockLockMessage) error
}

type ShopOrderService struct {
	shopRepository   *repository.ShopRepository
	orderRepository  *repository.ShopOrderRepository
	stockStore       *cache.ShopStockStore
	stockPublisher   ShopStockLockPublisher
	orderNoGenerator func() (string, error)
}

type CreateShopOrderInput struct {
	UserID    string
	ProductID uint64
	SkuID     uint64
	BuyCount  int
	RequestID string
}

func NewShopOrderService(
	shopRepository *repository.ShopRepository,
	orderRepository *repository.ShopOrderRepository,
	stockStore *cache.ShopStockStore,
	stockPublisher ShopStockLockPublisher,
) *ShopOrderService {
	return &ShopOrderService{
		shopRepository:   shopRepository,
		orderRepository:  orderRepository,
		stockStore:       stockStore,
		stockPublisher:   stockPublisher,
		orderNoGenerator: generateShopOrderNo,
	}
}

func (s *ShopOrderService) CreateOrder(ctx context.Context, input CreateShopOrderInput) (*domain.WebShopOrderItem, error) {
	input = normalizeCreateOrderInput(input)
	if err := validateCreateOrderInput(input); err != nil {
		return nil, err
	}
	if s.stockPublisher == nil {
		return nil, &BusinessError{Info: "抢购通道暂不可用，请稍后重试"}
	}

	skuInfo, err := s.shopRepository.FindPurchaseSKU(ctx, input.ProductID, input.SkuID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "商品不存在或已下架"}
	}
	if err != nil {
		return nil, err
	}
	if err := validatePurchaseSKU(skuInfo, input.BuyCount); err != nil {
		return nil, err
	}

	if err := s.stockStore.InitSKUStockIfAbsent(ctx, input.SkuID, skuInfo.AvailableStock); err != nil {
		return nil, err
	}

	orderNo, err := s.orderNoGenerator()
	if err != nil {
		return nil, err
	}

	deductResult, existingOrderNo, err := s.preDeductStock(ctx, input, skuInfo, orderNo)
	if err != nil {
		return nil, err
	}
	switch deductResult {
	case cache.StockPreDeductRepeated:
		return s.getRepeatedOrder(ctx, input.UserID, existingOrderNo)
	case cache.StockPreDeductInsufficient:
		return nil, &BusinessError{Info: "库存不足"}
	case cache.StockPreDeductSuccess:
	default:
		return nil, &BusinessError{Info: "抢购失败，请稍后重试"}
	}

	totalAmount := roundMoney(skuInfo.Price * float64(input.BuyCount))
	if err := s.orderRepository.CreatePendingOrder(ctx, repository.CreateShopOrderData{
		OrderNo:       orderNo,
		UserID:        input.UserID,
		ProductID:     skuInfo.ProductID,
		SkuID:         skuInfo.SkuID,
		ProductName:   skuInfo.ProductName,
		SkuName:       normalizeOrderSkuName(skuInfo.SkuName),
		CoverURL:      skuInfo.CoverURL,
		Price:         skuInfo.Price,
		BuyCount:      input.BuyCount,
		TotalAmount:   totalAmount,
		ExpireMinutes: defaultOrderExpireMinutes,
	}); err != nil {
		_ = s.stockStore.CompensateStock(context.Background(), input.SkuID, input.BuyCount)
		_ = s.stockStore.DeleteRequest(context.Background(), input.UserID, input.RequestID)
		return nil, err
	}

	message := mq.ShopStockLockMessage{
		MessageID: orderNo,
		OrderNo:   orderNo,
		UserID:    input.UserID,
		ProductID: input.ProductID,
		SkuID:     input.SkuID,
		BuyCount:  input.BuyCount,
	}
	if err := s.stockPublisher.PublishShopStockLockMessage(ctx, message); err != nil {
		_ = s.orderRepository.MarkOrderStockFailed(context.Background(), orderNo)
		_ = s.stockStore.CompensateStock(context.Background(), input.SkuID, input.BuyCount)
		_ = s.stockStore.DeleteRequest(context.Background(), input.UserID, input.RequestID)
		log.Printf("publish shop stock lock message failed: orderNo=%s err=%v", orderNo, err)
		return nil, &BusinessError{Info: "抢购请求提交失败，请稍后重试"}
	}

	return s.GetOrderDetail(ctx, input.UserID, orderNo)
}

func (s *ShopOrderService) GetOrderDetail(ctx context.Context, userID string, orderNo string) (*domain.WebShopOrderItem, error) {
	userID = strings.TrimSpace(userID)
	orderNo = strings.TrimSpace(orderNo)
	if userID == "" || orderNo == "" {
		return nil, &BusinessError{Info: "参数错误"}
	}

	item, err := s.orderRepository.FindOrderByNoAndUser(ctx, orderNo, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "订单不存在"}
	}
	if err != nil {
		return nil, err
	}
	fillWebShopOrderItem(item)
	return item, nil
}

func (s *ShopOrderService) LoadOrder(ctx context.Context, userID string, pageNo int, pageSize int) (domain.PaginationResult[domain.WebShopOrderItem], error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.PaginationResult[domain.WebShopOrderItem]{}, &BusinessError{Info: "请先登录"}
	}
	if pageNo <= 0 {
		pageNo = defaultOrderPageNo
	}
	if pageSize <= 0 {
		pageSize = defaultOrderPageSize
	}
	if pageSize > maxOrderPageSize {
		pageSize = maxOrderPageSize
	}

	list, totalCount, err := s.orderRepository.ListOrdersByUser(ctx, userID, pageNo, pageSize)
	if err != nil {
		return domain.PaginationResult[domain.WebShopOrderItem]{}, err
	}
	for index := range list {
		fillWebShopOrderItem(&list[index])
	}
	return domain.NewPaginationResult(list, totalCount, pageNo, pageSize), nil
}

func (s *ShopOrderService) HandleShopStockLockMessage(ctx context.Context, message mq.ShopStockLockMessage) error {
	if strings.TrimSpace(message.OrderNo) == "" || message.SkuID == 0 || message.BuyCount <= 0 {
		return nil
	}

	err := s.orderRepository.LockOrderStock(ctx, message.OrderNo, message.SkuID, message.BuyCount)
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrOrderStockInsufficient) {
		if refreshErr := s.refreshSKUStockFromDB(ctx, message.ProductID, message.SkuID); refreshErr != nil {
			log.Printf("refresh sku redis stock after mysql insufficient failed: skuID=%d err=%v", message.SkuID, refreshErr)
		}
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("discard shop stock lock message because order or sku not found: orderNo=%s skuID=%d", message.OrderNo, message.SkuID)
		return nil
	}
	return err
}

func (s *ShopOrderService) preDeductStock(ctx context.Context, input CreateShopOrderInput, skuInfo *repository.PurchaseSKUInfo, orderNo string) (cache.StockPreDeductResult, string, error) {
	result, existingOrderNo, err := s.stockStore.PreDeductStock(ctx, input.UserID, input.RequestID, input.SkuID, orderNo, input.BuyCount)
	if err != nil || result != cache.StockPreDeductMissing {
		return result, existingOrderNo, err
	}

	if err := s.stockStore.ResetSKUStock(ctx, input.SkuID, skuInfo.AvailableStock); err != nil {
		return cache.StockPreDeductInsufficient, "", err
	}
	return s.stockStore.PreDeductStock(ctx, input.UserID, input.RequestID, input.SkuID, orderNo, input.BuyCount)
}

func (s *ShopOrderService) getRepeatedOrder(ctx context.Context, userID string, orderNo string) (*domain.WebShopOrderItem, error) {
	if strings.TrimSpace(orderNo) == "" {
		return nil, &BusinessError{Info: "订单处理中，请稍后查看"}
	}
	item, err := s.orderRepository.FindOrderByNoAndUser(ctx, orderNo, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "订单处理中，请稍后查看"}
	}
	if err != nil {
		return nil, err
	}
	fillWebShopOrderItem(item)
	return item, nil
}

func (s *ShopOrderService) refreshSKUStockFromDB(ctx context.Context, productID uint64, skuID uint64) error {
	if productID == 0 || skuID == 0 {
		return nil
	}
	skuInfo, err := s.shopRepository.FindPurchaseSKU(ctx, productID, skuID)
	if err != nil {
		return err
	}
	return s.stockStore.ResetSKUStock(ctx, skuID, skuInfo.AvailableStock)
}

func normalizeCreateOrderInput(input CreateShopOrderInput) CreateShopOrderInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.RequestID == "" {
		input.RequestID = fmt.Sprintf("server-%d", time.Now().UnixNano())
	}
	if input.BuyCount <= 0 {
		input.BuyCount = 1
	}
	return input
}

func validateCreateOrderInput(input CreateShopOrderInput) error {
	if input.UserID == "" {
		return &BusinessError{Info: "请先登录"}
	}
	if input.ProductID == 0 || input.SkuID == 0 || input.BuyCount <= 0 {
		return &BusinessError{Info: "参数错误"}
	}
	if input.BuyCount > 99 {
		return &BusinessError{Info: "单次购买数量不能超过 99 件"}
	}
	return nil
}

func validatePurchaseSKU(info *repository.PurchaseSKUInfo, buyCount int) error {
	if info == nil {
		return &BusinessError{Info: "商品不存在或已下架"}
	}
	if info.ProductStatus != domain.ProductStatusOnShelf {
		return &BusinessError{Info: "商品已下架"}
	}
	if info.SaleStartTime != nil && time.Now().Before(*info.SaleStartTime) {
		return &BusinessError{Info: "商品尚未开售"}
	}
	if info.SkuStatus != domain.ProductStatusOnShelf {
		return &BusinessError{Info: "该规格已停用"}
	}
	if info.Price <= 0 {
		return &BusinessError{Info: "商品价格异常"}
	}
	if info.AvailableStock < buyCount {
		return &BusinessError{Info: "库存不足"}
	}
	if info.LimitPerUser > 0 && buyCount > info.LimitPerUser {
		return &BusinessError{Info: fmt.Sprintf("该规格每人限购 %d 件", info.LimitPerUser)}
	}
	return nil
}

func fillWebShopOrderItem(item *domain.WebShopOrderItem) {
	if item == nil {
		return
	}
	item.SkuName = normalizeOrderSkuName(item.SkuName)
	item.OrderStatusName = orderStatusName(item.OrderStatus)
	item.PriceText = formatOrderMoney(item.Price)
	item.TotalAmountText = formatOrderMoney(item.TotalAmount)
}

func orderStatusName(status int) string {
	switch status {
	case domain.OrderStatusStockLocking:
		return "库存锁定中"
	case domain.OrderStatusWaitPay:
		return "待支付"
	case domain.OrderStatusStockFailed:
		return "抢购失败"
	default:
		return "未知状态"
	}
}

func normalizeOrderSkuName(skuName string) string {
	skuName = strings.TrimSpace(skuName)
	if skuName == "" {
		return defaultOrderSkuName
	}
	return skuName
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func formatOrderMoney(value float64) string {
	return "¥" + strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func generateShopOrderNo() (string, error) {
	randomPart, err := randomDigitString(12)
	if err != nil {
		return "", err
	}
	return time.Now().Format("20060102150405") + randomPart, nil
}

func randomDigitString(length int) (string, error) {
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		number, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		builder.WriteByte(byte('0' + number.Int64()))
	}
	return builder.String(), nil
}
