package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

var ErrOrderStockInsufficient = errors.New("order stock insufficient")

var ErrOrderStockLockSkipped = errors.New("order stock lock skipped")

var ErrOrderCannotClose = errors.New("order cannot close")

type ShopOrderRepository struct {
	db *gorm.DB
}

type CreateShopOrderData struct {
	OrderNo        string
	UserID         string
	ProductID      uint64
	SkuID          uint64
	ProductName    string
	SkuName        string
	CoverURL       string
	Price          float64
	BuyCount       int
	TotalAmount    float64
	ExpireMinutes  int
	MessagePayload string
}

type CloseShopOrderData struct {
	OrderNo      string
	UserID       string
	TargetStatus int
	Now          time.Time
}

type CloseShopOrderResult struct {
	OrderNo             string
	ProductID           uint64
	SkuID               uint64
	Quantity            int
	Closed              bool
	ReleasedLockedStock bool
}

func NewShopOrderRepository(db *gorm.DB) *ShopOrderRepository {
	return &ShopOrderRepository{
		db: db,
	}
}

func (r *ShopOrderRepository) CreatePendingOrder(ctx context.Context, data CreateShopOrderData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order := &domain.ShopOrder{
			OrderNo:     data.OrderNo,
			UserID:      data.UserID,
			OrderStatus: domain.OrderStatusStockLocking,
			PayStatus:   domain.PayStatusUnpaid,
			TotalAmount: data.TotalAmount,
			PayAmount:   data.TotalAmount,
			ExpireTime:  nowAddMinutes(data.ExpireMinutes),
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		item := &domain.ShopOrderItem{
			OrderID:     order.OrderID,
			OrderNo:     data.OrderNo,
			ProductID:   data.ProductID,
			SkuID:       data.SkuID,
			ProductName: data.ProductName,
			SkuName:     data.SkuName,
			CoverURL:    data.CoverURL,
			Price:       data.Price,
			Quantity:    data.BuyCount,
			TotalAmount: data.TotalAmount,
		}
		if err := tx.Create(item).Error; err != nil {
			return err
		}

		nextRetryTime := time.Now()
		payload := data.MessagePayload
		if payload == "" {
			payload = "{}"
		}
		message := &domain.ShopOrderMessage{
			OrderNo:       data.OrderNo,
			MessageType:   domain.OrderMessageTypeStockLock,
			MessageStatus: domain.OrderMessageStatusWaitPublish,
			Payload:       payload,
			NextRetryTime: &nextRetryTime,
		}
		return tx.Create(message).Error
	})
}

func (r *ShopOrderRepository) MarkStockLockMessagePublished(ctx context.Context, orderNo string, nextRetryTime time.Time) error {
	return r.db.WithContext(ctx).Model(&domain.ShopOrderMessage{}).
		Where("order_no = ? AND message_type = ? AND message_status IN ?", orderNo, domain.OrderMessageTypeStockLock, []int{
			domain.OrderMessageStatusWaitPublish,
			domain.OrderMessageStatusPublished,
		}).
		Updates(map[string]any{
			"message_status":  domain.OrderMessageStatusPublished,
			"retry_count":     gorm.Expr("retry_count + ?", 1),
			"next_retry_time": nextRetryTime,
			"last_error":      "",
		}).Error
}

func (r *ShopOrderRepository) MarkStockLockMessageConsumedSuccess(ctx context.Context, orderNo string) error {
	return r.db.WithContext(ctx).Model(&domain.ShopOrderMessage{}).
		Where("order_no = ? AND message_type = ?", orderNo, domain.OrderMessageTypeStockLock).
		Updates(map[string]any{
			"message_status":  domain.OrderMessageStatusConsumedSuccess,
			"next_retry_time": nil,
			"last_error":      "",
		}).Error
}

func (r *ShopOrderRepository) MarkStockLockMessageConsumedFailed(ctx context.Context, orderNo string, lastError string) error {
	return r.db.WithContext(ctx).Model(&domain.ShopOrderMessage{}).
		Where("order_no = ? AND message_type = ?", orderNo, domain.OrderMessageTypeStockLock).
		Updates(map[string]any{
			"message_status":  domain.OrderMessageStatusConsumedFailed,
			"next_retry_time": nil,
			"last_error":      trimMessageError(lastError),
		}).Error
}

func (r *ShopOrderRepository) DelayStockLockMessageRetryByID(ctx context.Context, messageID uint64, nextRetryTime time.Time, lastError string, dead bool) error {
	var status any = gorm.Expr("message_status")
	if dead {
		status = domain.OrderMessageStatusDead
	}
	return r.db.WithContext(ctx).Model(&domain.ShopOrderMessage{}).
		Where("message_id = ? AND message_type = ?", messageID, domain.OrderMessageTypeStockLock).
		Updates(map[string]any{
			"message_status":  status,
			"retry_count":     gorm.Expr("retry_count + ?", 1),
			"next_retry_time": nextRetryTime,
			"last_error":      trimMessageError(lastError),
		}).Error
}

func (r *ShopOrderRepository) DelayStockLockMessageRetryByOrderNo(ctx context.Context, orderNo string, nextRetryTime time.Time, lastError string, dead bool) error {
	var status any = gorm.Expr("message_status")
	if dead {
		status = domain.OrderMessageStatusDead
	}
	return r.db.WithContext(ctx).Model(&domain.ShopOrderMessage{}).
		Where("order_no = ? AND message_type = ?", orderNo, domain.OrderMessageTypeStockLock).
		Updates(map[string]any{
			"message_status":  status,
			"retry_count":     gorm.Expr("retry_count + ?", 1),
			"next_retry_time": nextRetryTime,
			"last_error":      trimMessageError(lastError),
		}).Error
}

func (r *ShopOrderRepository) ListStockLockMessagesForRetry(ctx context.Context, limit int) ([]domain.ShopOrderMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	var list []domain.ShopOrderMessage
	err := r.db.WithContext(ctx).
		Where("message_type = ? AND message_status IN ?", domain.OrderMessageTypeStockLock, []int{
			domain.OrderMessageStatusWaitPublish,
			domain.OrderMessageStatusPublished,
		}).
		Where("next_retry_time IS NULL OR next_retry_time <= ?", time.Now()).
		Order("message_id asc").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (r *ShopOrderRepository) FindStockLockMessageByOrderNo(ctx context.Context, orderNo string) (*domain.ShopOrderMessage, error) {
	var message domain.ShopOrderMessage
	err := r.db.WithContext(ctx).
		Where("order_no = ? AND message_type = ?", orderNo, domain.OrderMessageTypeStockLock).
		Take(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *ShopOrderRepository) LockOrderStock(ctx context.Context, orderNo string, skuID uint64, buyCount int) error {
	stockInsufficient := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order domain.ShopOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", orderNo).
			Take(&order).Error; err != nil {
			return err
		}
		if order.OrderStatus == domain.OrderStatusWaitPay {
			return nil
		}
		if order.OrderStatus != domain.OrderStatusStockLocking {
			return ErrOrderStockLockSkipped
		}

		var flowCount int64
		if err := tx.Model(&domain.ShopStockFlow{}).
			Where("sku_id = ? AND order_no = ? AND change_type = ?", skuID, orderNo, domain.StockFlowTypeLock).
			Count(&flowCount).Error; err != nil {
			return err
		}
		if flowCount > 0 {
			return tx.Model(&domain.ShopOrder{}).
				Where("order_no = ?", orderNo).
				Update("order_status", domain.OrderStatusWaitPay).Error
		}

		var sku domain.ShopSKU
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("sku_id = ?", skuID).
			Take(&sku).Error; err != nil {
			return err
		}

		beforeLocked := sku.LockedStock
		beforeSold := sku.SoldStock
		result := tx.Model(&domain.ShopSKU{}).
			Where("sku_id = ? AND status = ? AND total_stock - locked_stock - sold_stock >= ?", skuID, domain.ProductStatusOnShelf, buyCount).
			Updates(map[string]any{
				"locked_stock": gorm.Expr("locked_stock + ?", buyCount),
				"version":      gorm.Expr("version + ?", 1),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err := markOrderStockFailed(tx, orderNo); err != nil {
				return err
			}
			stockInsufficient = true
			return nil
		}

		flow := &domain.ShopStockFlow{
			SkuID:             skuID,
			OrderNo:           orderNo,
			ChangeType:        domain.StockFlowTypeLock,
			ChangeCount:       buyCount,
			BeforeLockedStock: beforeLocked,
			AfterLockedStock:  beforeLocked + buyCount,
			BeforeSoldStock:   beforeSold,
			AfterSoldStock:    beforeSold,
		}
		if err := tx.Create(flow).Error; err != nil {
			return err
		}

		return tx.Model(&domain.ShopOrder{}).
			Where("order_no = ?", orderNo).
			Update("order_status", domain.OrderStatusWaitPay).Error
	})
	if err != nil {
		return err
	}
	if stockInsufficient {
		return ErrOrderStockInsufficient
	}
	return nil
}

func (r *ShopOrderRepository) CloseUnpaidOrder(ctx context.Context, data CloseShopOrderData) (CloseShopOrderResult, error) {
	result := CloseShopOrderResult{OrderNo: data.OrderNo}
	if data.Now.IsZero() {
		data.Now = time.Now()
	}
	if data.TargetStatus != domain.OrderStatusCanceled && data.TargetStatus != domain.OrderStatusTimeout {
		return result, ErrOrderCannotClose
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order domain.ShopOrder
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", data.OrderNo)
		if data.UserID != "" {
			query = query.Where("user_id = ?", data.UserID)
		}
		if err := query.Take(&order).Error; err != nil {
			return err
		}

		result.OrderNo = order.OrderNo
		switch order.OrderStatus {
		case domain.OrderStatusCanceled, domain.OrderStatusTimeout:
			return nil
		case domain.OrderStatusStockLocking, domain.OrderStatusWaitPay:
		default:
			return ErrOrderCannotClose
		}
		if order.PayStatus != domain.PayStatusUnpaid {
			return ErrOrderCannotClose
		}
		if data.TargetStatus == domain.OrderStatusTimeout && order.ExpireTime.After(data.Now) {
			return ErrOrderCannotClose
		}

		var item domain.ShopOrderItem
		if err := tx.Where("order_no = ?", order.OrderNo).Take(&item).Error; err != nil {
			return err
		}
		result.ProductID = item.ProductID
		result.SkuID = item.SkuID
		result.Quantity = item.Quantity

		var lockFlowCount int64
		if err := tx.Model(&domain.ShopStockFlow{}).
			Where("sku_id = ? AND order_no = ? AND change_type = ?", item.SkuID, order.OrderNo, domain.StockFlowTypeLock).
			Count(&lockFlowCount).Error; err != nil {
			return err
		}
		var unlockFlowCount int64
		if err := tx.Model(&domain.ShopStockFlow{}).
			Where("sku_id = ? AND order_no = ? AND change_type = ?", item.SkuID, order.OrderNo, domain.StockFlowTypeUnlock).
			Count(&unlockFlowCount).Error; err != nil {
			return err
		}

		if lockFlowCount > 0 && unlockFlowCount == 0 {
			if err := closeOrderLockedStock(tx, order.OrderNo, item); err != nil {
				return err
			}
			result.ReleasedLockedStock = item.Quantity > 0
		}

		updateResult := tx.Model(&domain.ShopOrder{}).
			Where("order_no = ? AND order_status IN ?", order.OrderNo, []int{
				domain.OrderStatusStockLocking,
				domain.OrderStatusWaitPay,
			}).
			Updates(map[string]any{
				"order_status": data.TargetStatus,
				"cancel_time":  data.Now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		result.Closed = updateResult.RowsAffected > 0
		return nil
	})
	return result, err
}

func (r *ShopOrderRepository) ListExpiredUnpaidOrderNos(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}
	var orderNos []string
	err := r.db.WithContext(ctx).
		Model(&domain.ShopOrder{}).
		Where("pay_status = ?", domain.PayStatusUnpaid).
		Where("order_status IN ?", []int{
			domain.OrderStatusStockLocking,
			domain.OrderStatusWaitPay,
		}).
		Where("expire_time <= ?", now).
		Order("order_id asc").
		Limit(limit).
		Pluck("order_no", &orderNos).Error
	return orderNos, err
}

func (r *ShopOrderRepository) MarkOrderStockFailed(ctx context.Context, orderNo string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return markOrderStockFailed(tx, orderNo)
	})
}

func (r *ShopOrderRepository) FindOrderByNo(ctx context.Context, orderNo string) (*domain.ShopOrder, error) {
	var order domain.ShopOrder
	err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).Take(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *ShopOrderRepository) FindOrderByNoAndUser(ctx context.Context, orderNo string, userID string) (*domain.WebShopOrderItem, error) {
	var item domain.WebShopOrderItem
	err := r.orderListQuery(ctx).
		Where("so.order_no = ? AND so.user_id = ?", orderNo, userID).
		Take(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ShopOrderRepository) ListOrdersByUser(ctx context.Context, userID string, pageNo int, pageSize int) ([]domain.WebShopOrderItem, int64, error) {
	var totalCount int64
	countDB := r.db.WithContext(ctx).Table("shop_order so").Where("so.user_id = ?", userID)
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.WebShopOrderItem
	offset := (pageNo - 1) * pageSize
	err := r.orderListQuery(ctx).
		Where("so.user_id = ?", userID).
		Order("so.create_time desc").
		Offset(offset).
		Limit(pageSize).
		Scan(&list).Error
	return list, totalCount, err
}

func (r *ShopOrderRepository) orderListQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("shop_order so").
		Joins("JOIN shop_order_item soi ON soi.order_no = so.order_no").
		Select(`
			so.order_id,
			so.order_no,
			so.user_id,
			so.order_status,
			so.pay_status,
			COALESCE(so.total_amount, 0) AS total_amount,
			COALESCE(so.pay_amount, 0) AS pay_amount,
			soi.product_id,
			soi.sku_id,
			COALESCE(soi.product_name, '') AS product_name,
			COALESCE(soi.sku_name, '') AS sku_name,
			COALESCE(soi.cover_url, '') AS cover_url,
			COALESCE(soi.price, 0) AS price,
			COALESCE(soi.quantity, 0) AS quantity,
			COALESCE(DATE_FORMAT(so.expire_time, '%Y-%m-%d %H:%i:%s'), '') AS expire_time,
			COALESCE(DATE_FORMAT(so.create_time, '%Y-%m-%d %H:%i:%s'), '') AS create_time,
			COALESCE(DATE_FORMAT(so.update_time, '%Y-%m-%d %H:%i:%s'), '') AS update_time
		`)
}

func markOrderStockFailed(tx *gorm.DB, orderNo string) error {
	return tx.Model(&domain.ShopOrder{}).
		Where("order_no = ? AND order_status = ?", orderNo, domain.OrderStatusStockLocking).
		Update("order_status", domain.OrderStatusStockFailed).Error
}

func closeOrderLockedStock(tx *gorm.DB, orderNo string, item domain.ShopOrderItem) error {
	var sku domain.ShopSKU
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sku_id = ?", item.SkuID).
		Take(&sku).Error; err != nil {
		return err
	}

	afterLocked := sku.LockedStock - item.Quantity
	if afterLocked < 0 {
		afterLocked = 0
	}

	updateResult := tx.Model(&domain.ShopSKU{}).
		Where("sku_id = ?", item.SkuID).
		Updates(map[string]any{
			"locked_stock": gorm.Expr("GREATEST(locked_stock - ?, 0)", item.Quantity),
			"version":      gorm.Expr("version + ?", 1),
		})
	if updateResult.Error != nil {
		return updateResult.Error
	}
	if updateResult.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	flow := &domain.ShopStockFlow{
		SkuID:             item.SkuID,
		OrderNo:           orderNo,
		ChangeType:        domain.StockFlowTypeUnlock,
		ChangeCount:       item.Quantity,
		BeforeLockedStock: sku.LockedStock,
		AfterLockedStock:  afterLocked,
		BeforeSoldStock:   sku.SoldStock,
		AfterSoldStock:    sku.SoldStock,
	}
	return tx.Create(flow).Error
}

func nowAddMinutes(minutes int) time.Time {
	if minutes <= 0 {
		minutes = 15
	}
	return time.Now().Add(time.Duration(minutes) * time.Minute)
}

func trimMessageError(message string) string {
	if len(message) <= 500 {
		return message
	}
	return message[:500]
}
