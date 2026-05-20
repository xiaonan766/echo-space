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

type ShopOrderRepository struct {
	db *gorm.DB
}

type CreateShopOrderData struct {
	OrderNo       string
	UserID        string
	ProductID     uint64
	SkuID         uint64
	ProductName   string
	SkuName       string
	CoverURL      string
	Price         float64
	BuyCount      int
	TotalAmount   float64
	ExpireMinutes int
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
		return tx.Create(item).Error
	})
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
		if order.OrderStatus == domain.OrderStatusWaitPay || order.OrderStatus == domain.OrderStatusStockFailed {
			return nil
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

func (r *ShopOrderRepository) MarkOrderStockFailed(ctx context.Context, orderNo string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return markOrderStockFailed(tx, orderNo)
	})
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

func nowAddMinutes(minutes int) time.Time {
	if minutes <= 0 {
		minutes = 15
	}
	return time.Now().Add(time.Duration(minutes) * time.Minute)
}
