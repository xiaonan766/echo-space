package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type ShopCabinetRepository struct {
	db *gorm.DB
}

type PeripheralCabinetQuery struct {
	UserID        string
	PageNo        int
	PageSize      int
	IncludeHidden bool
}

type PaidCabinetSKU struct {
	ProductID uint64 `gorm:"column:product_id"`
	SkuID     uint64 `gorm:"column:sku_id"`
}

func NewShopCabinetRepository(db *gorm.DB) *ShopCabinetRepository {
	return &ShopCabinetRepository{db: db}
}

func (r *ShopCabinetRepository) GetCabinetVisible(ctx context.Context, userID string) (bool, error) {
	var row struct {
		ShopCabinetVisible int `gorm:"column:shop_cabinet_visible"`
	}
	err := r.db.WithContext(ctx).
		Table("user_info").
		Select("shop_cabinet_visible").
		Where("user_id = ?", userID).
		Take(&row).Error
	if err != nil {
		return false, err
	}
	return row.ShopCabinetVisible == 1, nil
}

func (r *ShopCabinetRepository) UpdateCabinetVisible(ctx context.Context, userID string, visible bool) error {
	visibleValue := 0
	if visible {
		visibleValue = 1
	}
	return r.db.WithContext(ctx).
		Table("user_info").
		Where("user_id = ?", userID).
		Update("shop_cabinet_visible", visibleValue).Error
}

func (r *ShopCabinetRepository) ListPeripheralCabinet(ctx context.Context, query PeripheralCabinetQuery) ([]domain.WebPeripheralCabinetItem, int64, error) {
	countQuery := r.basePeripheralCabinetQuery(ctx, query).
		Select("soi.sku_id").
		Group("soi.sku_id")

	var totalCount int64
	if err := r.db.WithContext(ctx).Table("(?) AS cabinet", countQuery).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.WebPeripheralCabinetItem
	offset := (query.PageNo - 1) * query.PageSize
	err := r.basePeripheralCabinetQuery(ctx, query).
		Select(`
			MAX(soi.product_id) AS product_id,
			soi.sku_id AS sku_id,
			COALESCE(MAX(soi.product_name), '') AS product_name,
			COALESCE(MAX(soi.sku_name), '') AS sku_name,
			COALESCE(MAX(soi.cover_url), '') AS cover_url,
			COALESCE(SUM(soi.quantity), 0) AS owned_quantity,
			COUNT(DISTINCT so.order_no) AS order_count,
			COALESCE(DATE_FORMAT(MAX(COALESCE(so.pay_time, so.update_time, so.create_time)), '%Y-%m-%d %H:%i:%s'), '') AS latest_buy_time,
			CASE WHEN MAX(CASE WHEN schi.id IS NULL THEN 0 ELSE 1 END) > 0 THEN 'true' ELSE 'false' END AS hidden
		`).
		Group("soi.sku_id").
		Order("MAX(COALESCE(so.pay_time, so.update_time, so.create_time)) DESC, soi.sku_id DESC").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []domain.WebPeripheralCabinetItem{}
	}
	return list, totalCount, nil
}

func (r *ShopCabinetRepository) FindPaidCabinetSKU(ctx context.Context, userID string, skuID uint64) (*PaidCabinetSKU, error) {
	var row PaidCabinetSKU
	err := r.db.WithContext(ctx).
		Table("shop_order so").
		Joins("JOIN shop_order_item soi ON soi.order_no = so.order_no").
		Select("soi.product_id, soi.sku_id").
		Where("so.user_id = ? AND soi.sku_id = ?", userID, skuID).
		Where("so.order_status = ? AND so.pay_status = ?", domain.OrderStatusPaid, domain.PayStatusPaid).
		Order("COALESCE(so.pay_time, so.update_time, so.create_time) DESC, so.order_id DESC").
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ShopCabinetRepository) HideCabinetItem(ctx context.Context, userID string, productID uint64, skuID uint64, hideTime time.Time) error {
	item := &domain.ShopCabinetHiddenItem{
		UserID:    userID,
		ProductID: productID,
		SkuID:     skuID,
		HideTime:  hideTime,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "sku_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"product_id": productID,
				"hide_time":  hideTime,
			}),
		}).
		Create(item).Error
}

func (r *ShopCabinetRepository) ShowCabinetItem(ctx context.Context, userID string, skuID uint64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND sku_id = ?", userID, skuID).
		Delete(&domain.ShopCabinetHiddenItem{}).Error
}

func (r *ShopCabinetRepository) basePeripheralCabinetQuery(ctx context.Context, query PeripheralCabinetQuery) *gorm.DB {
	db := r.db.WithContext(ctx).
		Table("shop_order so").
		Joins("JOIN shop_order_item soi ON soi.order_no = so.order_no").
		Joins("LEFT JOIN shop_cabinet_hidden_item schi ON schi.user_id = so.user_id AND schi.sku_id = soi.sku_id").
		Where("so.user_id = ?", query.UserID).
		Where("so.order_status = ? AND so.pay_status = ?", domain.OrderStatusPaid, domain.PayStatusPaid)
	if !query.IncludeHidden {
		db = db.Where("schi.id IS NULL")
	}
	return db
}
