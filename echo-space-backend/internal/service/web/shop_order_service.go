package web

import (
	"context"
	"crypto/rand"
	"encoding/json"
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
	orderMessageRetryDelay    = 30 * time.Second
	orderMessageRetryInterval = 5 * time.Second
	orderReservationInterval  = 30 * time.Second
	orderExpiredCloseInterval = 30 * time.Second
	orderRecoveryBatchSize    = 100
	maxOrderMessageRetryCount = 10
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

	message := mq.ShopStockLockMessage{
		MessageID: orderNo,
		OrderNo:   orderNo,
		UserID:    input.UserID,
		ProductID: input.ProductID,
		SkuID:     input.SkuID,
		BuyCount:  input.BuyCount,
	}
	messagePayload, err := json.Marshal(message)
	if err != nil {
		_ = s.stockStore.ReleaseReservation(context.Background(), orderNo)
		_ = s.stockStore.DeleteRequest(context.Background(), input.UserID, input.RequestID)
		return nil, err
	}

	totalAmount := roundMoney(skuInfo.Price * float64(input.BuyCount))
	if err := s.orderRepository.CreatePendingOrder(ctx, repository.CreateShopOrderData{
		OrderNo:        orderNo,
		UserID:         input.UserID,
		ProductID:      skuInfo.ProductID,
		SkuID:          skuInfo.SkuID,
		ProductName:    skuInfo.ProductName,
		SkuName:        normalizeOrderSkuName(skuInfo.SkuName),
		CoverURL:       skuInfo.CoverURL,
		Price:          skuInfo.Price,
		BuyCount:       input.BuyCount,
		TotalAmount:    totalAmount,
		ExpireMinutes:  defaultOrderExpireMinutes,
		MessagePayload: string(messagePayload),
	}); err != nil {
		_ = s.stockStore.ReleaseReservation(context.Background(), orderNo)
		_ = s.stockStore.DeleteRequest(context.Background(), input.UserID, input.RequestID)
		return nil, err
	}

	if err := s.stockStore.MarkReservationOrderCreated(context.Background(), orderNo); err != nil {
		log.Printf("mark stock reservation order created failed: orderNo=%s err=%v", orderNo, err)
	}

	if err := s.publishStockLockMessage(ctx, message); err != nil {
		_ = s.orderRepository.DelayStockLockMessageRetryByOrderNo(context.Background(), orderNo, nextOrderMessageRetryTime(), err.Error(), false)
		log.Printf("publish shop stock lock message will retry: orderNo=%s err=%v", orderNo, err)
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

func (s *ShopOrderService) CancelOrder(ctx context.Context, userID string, orderNo string) (*domain.WebShopOrderItem, error) {
	userID = strings.TrimSpace(userID)
	orderNo = strings.TrimSpace(orderNo)
	if userID == "" {
		return nil, &BusinessError{Info: "请先登录"}
	}
	if orderNo == "" {
		return nil, &BusinessError{Info: "参数错误"}
	}

	result, err := s.orderRepository.CloseUnpaidOrder(ctx, repository.CloseShopOrderData{
		OrderNo:      orderNo,
		UserID:       userID,
		TargetStatus: domain.OrderStatusCanceled,
		Now:          time.Now(),
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "订单不存在"}
	}
	if errors.Is(err, repository.ErrOrderCannotClose) {
		return nil, &BusinessError{Info: "当前订单状态不支持取消"}
	}
	if err != nil {
		return nil, err
	}

	s.syncClosedOrderStock(context.Background(), result)
	return s.GetOrderDetail(ctx, userID, orderNo)
}

func (s *ShopOrderService) PayOrder(ctx context.Context, userID string, orderNo string) (*domain.WebShopOrderItem, error) {
	userID = strings.TrimSpace(userID)
	orderNo = strings.TrimSpace(orderNo)
	if userID == "" {
		return nil, &BusinessError{Info: "请先登录"}
	}
	if orderNo == "" {
		return nil, &BusinessError{Info: "参数错误"}
	}

	payResult, err := s.orderRepository.PayOrderByCoin(ctx, userID, orderNo, time.Now())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "订单不存在"}
	}
	if errors.Is(err, repository.ErrOrderExpired) {
		s.syncClosedOrderStock(context.Background(), payResult.ClosedOrder)
		return nil, &BusinessError{Info: "订单已超时关闭，请重新下单"}
	}
	if errors.Is(err, repository.ErrOrderInsufficientCoin) {
		return nil, &BusinessError{Info: "硬币余额不足"}
	}
	if errors.Is(err, repository.ErrOrderCannotPay) {
		return nil, &BusinessError{Info: "当前订单状态暂不支持支付"}
	}
	if err != nil {
		return nil, err
	}

	if err := s.stockStore.MarkReservationPaid(context.Background(), orderNo); err != nil {
		log.Printf("mark stock reservation paid failed: orderNo=%s err=%v", orderNo, err)
	}

	item, err := s.GetOrderDetail(ctx, userID, orderNo)
	if err != nil {
		return nil, err
	}
	item.CurrentCoinCount = payResult.CurrentCoinCount
	return item, nil
}

func (s *ShopOrderService) HandleShopStockLockMessage(ctx context.Context, message mq.ShopStockLockMessage) error {
	if strings.TrimSpace(message.OrderNo) == "" || message.SkuID == 0 || message.BuyCount <= 0 {
		return nil
	}

	err := s.orderRepository.LockOrderStock(ctx, message.OrderNo, message.SkuID, message.BuyCount)
	if err == nil {
		if markErr := s.orderRepository.MarkStockLockMessageConsumedSuccess(ctx, message.OrderNo); markErr != nil {
			return markErr
		}
		if markErr := s.stockStore.MarkReservationLocked(ctx, message.OrderNo); markErr != nil {
			log.Printf("mark stock reservation locked failed: orderNo=%s err=%v", message.OrderNo, markErr)
		}
		return nil
	}
	if errors.Is(err, repository.ErrOrderStockLockSkipped) {
		_ = s.orderRepository.MarkStockLockMessageConsumedFailed(ctx, message.OrderNo, err.Error())
		_ = s.stockStore.ReleaseReservation(ctx, message.OrderNo)
		log.Printf("skip shop stock lock message because order is closed: orderNo=%s skuID=%d", message.OrderNo, message.SkuID)
		return nil
	}
	if errors.Is(err, repository.ErrOrderStockInsufficient) {
		_ = s.orderRepository.MarkStockLockMessageConsumedFailed(ctx, message.OrderNo, err.Error())
		if refreshErr := s.refreshSKUStockFromDB(ctx, message.ProductID, message.SkuID); refreshErr != nil {
			log.Printf("refresh sku redis stock after mysql insufficient failed: skuID=%d err=%v", message.SkuID, refreshErr)
		}
		if markErr := s.stockStore.MarkReservationReleased(ctx, message.OrderNo); markErr != nil {
			log.Printf("mark stock reservation released failed: orderNo=%s err=%v", message.OrderNo, markErr)
		}
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = s.orderRepository.MarkStockLockMessageConsumedFailed(ctx, message.OrderNo, err.Error())
		_ = s.stockStore.ReleaseReservation(ctx, message.OrderNo)
		log.Printf("discard shop stock lock message because order or sku not found: orderNo=%s skuID=%d", message.OrderNo, message.SkuID)
		return nil
	}
	return err
}

func (s *ShopOrderService) preDeductStock(ctx context.Context, input CreateShopOrderInput, skuInfo *repository.PurchaseSKUInfo, orderNo string) (cache.StockPreDeductResult, string, error) {
	reservation := cache.StockReservation{
		OrderNo:   orderNo,
		UserID:    input.UserID,
		ProductID: input.ProductID,
		SkuID:     input.SkuID,
		BuyCount:  input.BuyCount,
	}
	result, existingOrderNo, err := s.stockStore.PreDeductStock(ctx, input.UserID, input.RequestID, reservation)
	if err != nil || result != cache.StockPreDeductMissing {
		return result, existingOrderNo, err
	}

	if err := s.stockStore.ResetSKUStock(ctx, input.SkuID, skuInfo.AvailableStock); err != nil {
		return cache.StockPreDeductInsufficient, "", err
	}
	return s.stockStore.PreDeductStock(ctx, input.UserID, input.RequestID, reservation)
}

func (s *ShopOrderService) RetryStockLockMessages(ctx context.Context, limit int) {
	if s == nil || s.stockPublisher == nil {
		return
	}
	messages, err := s.orderRepository.ListStockLockMessagesForRetry(ctx, limit)
	if err != nil {
		log.Printf("list stock lock messages for retry failed: %v", err)
		return
	}
	for _, message := range messages {
		s.retryStockLockMessage(ctx, message)
	}
}

func (s *ShopOrderService) RecoverExpiredReservations(ctx context.Context, limit int64) {
	orderNos, err := s.stockStore.ListExpiredReservations(ctx, time.Now().Unix(), limit)
	if err != nil {
		log.Printf("list expired stock reservations failed: %v", err)
		return
	}
	for _, orderNo := range orderNos {
		s.recoverExpiredReservation(ctx, orderNo)
	}
}

func (s *ShopOrderService) CloseExpiredUnpaidOrders(ctx context.Context, limit int) {
	orderNos, err := s.orderRepository.ListExpiredUnpaidOrderNos(ctx, time.Now(), limit)
	if err != nil {
		log.Printf("list expired unpaid orders failed: %v", err)
		return
	}
	for _, orderNo := range orderNos {
		s.closeExpiredUnpaidOrder(ctx, orderNo)
	}
}

func (s *ShopOrderService) StartRecoveryTasks(ctx context.Context) {
	go s.runMessageRetryTask(ctx)
	go s.runReservationRecoveryTask(ctx)
	go s.runExpiredOrderCloseTask(ctx)
}

func (s *ShopOrderService) retryStockLockMessage(ctx context.Context, message domain.ShopOrderMessage) {
	if message.RetryCount >= maxOrderMessageRetryCount {
		_ = s.orderRepository.DelayStockLockMessageRetryByID(ctx, message.MessageID, time.Now(), "message retry count exceeded", true)
		return
	}

	var payload mq.ShopStockLockMessage
	if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
		_ = s.orderRepository.DelayStockLockMessageRetryByID(ctx, message.MessageID, time.Now(), err.Error(), true)
		return
	}
	if payload.MessageID == "" {
		payload.MessageID = payload.OrderNo
	}
	if err := s.publishStockLockMessage(ctx, payload); err != nil {
		dead := message.RetryCount+1 >= maxOrderMessageRetryCount
		_ = s.orderRepository.DelayStockLockMessageRetryByID(ctx, message.MessageID, nextOrderMessageRetryTime(), err.Error(), dead)
	}
}

func (s *ShopOrderService) recoverExpiredReservation(ctx context.Context, orderNo string) {
	reservation, ok, err := s.stockStore.GetReservation(ctx, orderNo)
	if err != nil {
		log.Printf("get stock reservation failed: orderNo=%s err=%v", orderNo, err)
		return
	}
	if !ok {
		_ = s.stockStore.MarkReservationReleased(ctx, orderNo)
		return
	}
	if reservation.Status == cache.StockReservationStatusReleased {
		_ = s.stockStore.MarkReservationReleased(ctx, orderNo)
		return
	}
	if reservation.Status == cache.StockReservationStatusLocked {
		_ = s.stockStore.MarkReservationLocked(ctx, orderNo)
		return
	}

	order, err := s.orderRepository.FindOrderByNo(ctx, orderNo)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = s.stockStore.ReleaseReservation(ctx, orderNo)
		return
	}
	if err != nil {
		log.Printf("find order for reservation recovery failed: orderNo=%s err=%v", orderNo, err)
		return
	}

	switch order.OrderStatus {
	case domain.OrderStatusWaitPay:
		_ = s.stockStore.MarkReservationLocked(ctx, orderNo)
	case domain.OrderStatusStockFailed:
		_ = s.stockStore.ReleaseReservation(ctx, orderNo)
	case domain.OrderStatusCanceled, domain.OrderStatusTimeout:
		s.releaseClosedOrderReservation(ctx, orderNo, reservation)
	case domain.OrderStatusStockLocking:
		s.recoverStockLockingOrder(ctx, orderNo)
	}
}

func (s *ShopOrderService) recoverStockLockingOrder(ctx context.Context, orderNo string) {
	message, err := s.orderRepository.FindStockLockMessageByOrderNo(ctx, orderNo)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = s.orderRepository.MarkOrderStockFailed(ctx, orderNo)
		_ = s.stockStore.ReleaseReservation(ctx, orderNo)
		return
	}
	if err != nil {
		log.Printf("find stock lock message for reservation recovery failed: orderNo=%s err=%v", orderNo, err)
		return
	}
	switch message.MessageStatus {
	case domain.OrderMessageStatusWaitPublish, domain.OrderMessageStatusPublished:
		return
	case domain.OrderMessageStatusConsumedSuccess:
		_ = s.stockStore.MarkReservationLocked(ctx, orderNo)
	case domain.OrderMessageStatusConsumedFailed, domain.OrderMessageStatusDead:
		_ = s.orderRepository.MarkOrderStockFailed(ctx, orderNo)
		_ = s.stockStore.ReleaseReservation(ctx, orderNo)
	}
}

func (s *ShopOrderService) publishStockLockMessage(ctx context.Context, message mq.ShopStockLockMessage) error {
	if s.stockPublisher == nil {
		return errors.New("shop stock lock publisher is nil")
	}
	if err := s.stockPublisher.PublishShopStockLockMessage(ctx, message); err != nil {
		return err
	}
	return s.orderRepository.MarkStockLockMessagePublished(ctx, message.OrderNo, nextOrderMessageRetryTime())
}

func (s *ShopOrderService) runMessageRetryTask(ctx context.Context) {
	ticker := time.NewTicker(orderMessageRetryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.RetryStockLockMessages(ctx, orderRecoveryBatchSize)
		case <-ctx.Done():
			return
		}
	}
}

func (s *ShopOrderService) runReservationRecoveryTask(ctx context.Context) {
	ticker := time.NewTicker(orderReservationInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.RecoverExpiredReservations(ctx, orderRecoveryBatchSize)
		case <-ctx.Done():
			return
		}
	}
}

func (s *ShopOrderService) runExpiredOrderCloseTask(ctx context.Context) {
	ticker := time.NewTicker(orderExpiredCloseInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.CloseExpiredUnpaidOrders(ctx, orderRecoveryBatchSize)
		case <-ctx.Done():
			return
		}
	}
}

func (s *ShopOrderService) closeExpiredUnpaidOrder(ctx context.Context, orderNo string) {
	result, err := s.orderRepository.CloseUnpaidOrder(ctx, repository.CloseShopOrderData{
		OrderNo:      orderNo,
		TargetStatus: domain.OrderStatusTimeout,
		Now:          time.Now(),
	})
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, repository.ErrOrderCannotClose) {
		return
	}
	if err != nil {
		log.Printf("close expired unpaid order failed: orderNo=%s err=%v", orderNo, err)
		return
	}
	s.syncClosedOrderStock(context.Background(), result)
}

func (s *ShopOrderService) syncClosedOrderStock(ctx context.Context, result repository.CloseShopOrderResult) {
	if !result.Closed {
		return
	}
	if result.ReleasedLockedStock {
		if err := s.stockStore.ReleaseLockedReservation(ctx, result.OrderNo, result.SkuID, result.Quantity); err != nil {
			log.Printf("release locked stock reservation failed: orderNo=%s skuID=%d err=%v", result.OrderNo, result.SkuID, err)
		}
		return
	}
	if err := s.stockStore.ReleaseReservation(ctx, result.OrderNo); err != nil {
		log.Printf("release stock reservation failed: orderNo=%s err=%v", result.OrderNo, err)
	}
}

func (s *ShopOrderService) releaseClosedOrderReservation(ctx context.Context, orderNo string, reservation *cache.StockReservation) {
	if reservation == nil {
		_ = s.stockStore.ReleaseReservation(ctx, orderNo)
		return
	}
	if reservation.Status == cache.StockReservationStatusLocked {
		if err := s.stockStore.ReleaseLockedReservation(ctx, orderNo, reservation.SkuID, reservation.BuyCount); err != nil {
			log.Printf("release closed locked reservation failed: orderNo=%s skuID=%d err=%v", orderNo, reservation.SkuID, err)
		}
		return
	}
	if err := s.stockStore.ReleaseReservation(ctx, orderNo); err != nil {
		log.Printf("release closed reservation failed: orderNo=%s err=%v", orderNo, err)
	}
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
	case domain.OrderStatusPaid:
		return "已支付"
	case domain.OrderStatusCanceled:
		return "已取消"
	case domain.OrderStatusTimeout:
		return "已超时"
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

func nextOrderMessageRetryTime() time.Time {
	return time.Now().Add(orderMessageRetryDelay)
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
