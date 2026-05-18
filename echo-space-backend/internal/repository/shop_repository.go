package repository

import (
	"context"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const defaultPeripheralSkuName = "默认规格"

const priceChangeWaitDuration = 30 * time.Minute

type ShopRepository struct {
	db *gorm.DB
}

type PeripheralListQuery struct {
	PageNo           int
	PageSize         int
	ProductNameFuzzy string
	Status           *int
	SaleStatus       *int
}

type WebShopListQuery struct {
	PageNo   int
	PageSize int
	Keyword  string
}

type SavePeripheralData struct {
	ProductID       uint64
	ProductName     string
	CoverURL        string
	Description     string
	SaleStartTime   *time.Time
	Status          int
	RecommendStatus int
	Sort            int
	SkuList         []SavePeripheralSKUData
}

type SavePeripheralSKUData struct {
	SkuID      uint64
	SkuName    string
	Price      float64
	TotalStock int
	Status     int
}

func NewShopRepository(db *gorm.DB) *ShopRepository {
	return &ShopRepository{
		db: db,
	}
}

func (r *ShopRepository) ListPeripheralByPage(ctx context.Context, query PeripheralListQuery) ([]domain.AdminPeripheralItem, int64, error) {
	var totalCount int64
	countDB := r.applyPeripheralListFilter(r.db.WithContext(ctx).Table("shop_product sp"), query)
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.AdminPeripheralItem
	offset := (query.PageNo - 1) * query.PageSize
	listDB := r.applyPeripheralListFilter(r.db.WithContext(ctx).Table("shop_product sp"), query)
	err := listDB.
		Select(adminPeripheralSelectSQL()).
		Order("sp.sort desc").
		Order("sp.product_id desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, totalCount, nil
}

func (r *ShopRepository) FindPeripheralDetail(ctx context.Context, productID uint64) (*domain.AdminPeripheralItem, error) {
	var detail domain.AdminPeripheralItem
	err := r.applyPeripheralListFilter(r.db.WithContext(ctx).Table("shop_product sp"), PeripheralListQuery{}).
		Select(adminPeripheralSelectSQL()).
		Where("sp.product_id = ?", productID).
		Take(&detail).Error
	if err != nil {
		return nil, err
	}
	skuList, err := r.ListPeripheralSKU(ctx, productID)
	if err != nil {
		return nil, err
	}
	detail.SkuList = skuList
	return &detail, nil
}

func (r *ShopRepository) ListPeripheralSKU(ctx context.Context, productID uint64) ([]domain.AdminPeripheralSKU, error) {
	var list []domain.AdminPeripheralSKU
	err := r.db.WithContext(ctx).Table("shop_sku").
		Select(`
			sku_id,
			product_id,
			COALESCE(sku_name, '') AS sku_name,
			COALESCE(price, 0) AS price,
			COALESCE(total_stock, 0) AS total_stock,
			COALESCE(locked_stock, 0) AS locked_stock,
			COALESCE(sold_stock, 0) AS sold_stock,
			GREATEST(COALESCE(total_stock, 0) - COALESCE(locked_stock, 0) - COALESCE(sold_stock, 0), 0) AS available_stock,
			status
		`).
		Where("product_id = ?", productID).
		Order("sku_id asc").
		Scan(&list).Error
	return list, err
}

func (r *ShopRepository) ListRecommendedPeripheralForWeb(ctx context.Context, limit int) ([]domain.WebShopItem, error) {
	var list []domain.WebShopItem
	err := r.applyWebPeripheralListFilter(r.db.WithContext(ctx).Table("shop_product sp"), "").
		Select(webPeripheralSelectSQL()).
		Where("sp.recommend_status = ?", domain.RecommendStatusYes).
		Order("sp.sort desc").
		Order("sp.product_id desc").
		Limit(limit).
		Scan(&list).Error
	return list, err
}

func (r *ShopRepository) ListPeripheralForWeb(ctx context.Context, query WebShopListQuery) ([]domain.WebShopItem, int64, error) {
	var totalCount int64
	countDB := r.applyWebPeripheralListFilter(r.db.WithContext(ctx).Table("shop_product sp"), query.Keyword)
	if err := countDB.Distinct("sp.product_id").Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.WebShopItem
	offset := (query.PageNo - 1) * query.PageSize
	listDB := r.applyWebPeripheralListFilter(r.db.WithContext(ctx).Table("shop_product sp"), query.Keyword)
	err := listDB.
		Select(webPeripheralSelectSQL()).
		Order("sp.sort desc").
		Order("sp.product_id desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, totalCount, nil
}

func (r *ShopRepository) FindWebPeripheralDetail(ctx context.Context, productID uint64) (*domain.WebShopItem, error) {
	var detail domain.WebShopItem
	err := r.applyWebPeripheralListFilter(r.db.WithContext(ctx).Table("shop_product sp"), "").
		Select(webPeripheralSelectSQL()).
		Where("sp.product_id = ?", productID).
		Take(&detail).Error
	if err != nil {
		return nil, err
	}
	skuList, err := r.ListWebPeripheralSKU(ctx, productID)
	if err != nil {
		return nil, err
	}
	detail.SkuList = skuList
	return &detail, nil
}

func (r *ShopRepository) ListWebPeripheralSKU(ctx context.Context, productID uint64) ([]domain.WebShopSKU, error) {
	var list []domain.WebShopSKU
	err := r.db.WithContext(ctx).Table("shop_sku").
		Select(`
			sku_id,
			COALESCE(sku_name, '') AS sku_name,
			COALESCE(price, 0) AS price,
			COALESCE(total_stock, 0) AS total_stock,
			GREATEST(COALESCE(total_stock, 0) - COALESCE(locked_stock, 0) - COALESCE(sold_stock, 0), 0) AS available_stock,
			status,
			CASE
				WHEN COALESCE(total_stock, 0) - COALESCE(locked_stock, 0) - COALESCE(sold_stock, 0) <= 0 THEN 2
				ELSE 1
			END AS sale_status
		`).
		Where("product_id = ? AND status = ?", productID, domain.ProductStatusOnShelf).
		Order("sku_id asc").
		Scan(&list).Error
	return list, err
}

func (r *ShopRepository) SavePeripheral(ctx context.Context, data SavePeripheralData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if data.ProductID == 0 {
			product := &domain.ShopProduct{
				ProductType:     domain.ProductTypePeripheral,
				ProductName:     data.ProductName,
				CoverURL:        data.CoverURL,
				Description:     data.Description,
				Status:          data.Status,
				RecommendStatus: data.RecommendStatus,
				SaleStartTime:   data.SaleStartTime,
				Sort:            data.Sort,
			}
			if err := tx.Create(product).Error; err != nil {
				return err
			}
			for _, skuData := range data.SkuList {
				sku := &domain.ShopSKU{
					ProductID:  product.ProductID,
					SkuName:    skuData.SkuName,
					Price:      skuData.Price,
					TotalStock: skuData.TotalStock,
					Status:     skuData.Status,
				}
				if err := tx.Create(sku).Error; err != nil {
					return err
				}
			}
			return nil
		}

		var product domain.ShopProduct
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("product_id = ? AND product_type = ?", data.ProductID, domain.ProductTypePeripheral).
			Take(&product).Error; err != nil {
			return err
		}

		var existingSkuList []domain.ShopSKU
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("product_id = ?", data.ProductID).
			Find(&existingSkuList).Error; err != nil {
			return err
		}

		existingByID := make(map[uint64]domain.ShopSKU, len(existingSkuList))
		for _, sku := range existingSkuList {
			existingByID[sku.SkuID] = sku
		}

		priceChanged := false
		for _, skuData := range data.SkuList {
			if skuData.SkuID == 0 {
				priceChanged = true
				continue
			}
			sku, ok := existingByID[skuData.SkuID]
			if !ok {
				return gorm.ErrRecordNotFound
			}
			if skuData.TotalStock < sku.LockedStock+sku.SoldStock {
				return ErrStockLessThanOccupied
			}
			if isPriceChanged(sku.Price, skuData.Price) {
				priceChanged = true
			}
		}
		if priceChanged && !canChangePrice(product) {
			return ErrPriceChangeTooEarly
		}

		productUpdates := map[string]any{
			"product_name":     data.ProductName,
			"cover_url":        data.CoverURL,
			"description":      data.Description,
			"status":           data.Status,
			"recommend_status": data.RecommendStatus,
			"sale_start_time":  data.SaleStartTime,
			"sort":             data.Sort,
		}
		if product.Status == domain.ProductStatusOnShelf && data.Status == domain.ProductStatusOffShelf {
			productUpdates["last_off_shelf_time"] = time.Now()
		}

		if err := tx.Model(&domain.ShopProduct{}).
			Where("product_id = ? AND product_type = ?", data.ProductID, domain.ProductTypePeripheral).
			Updates(productUpdates).Error; err != nil {
			return err
		}

		for _, skuData := range data.SkuList {
			if skuData.SkuID == 0 {
				sku := &domain.ShopSKU{
					ProductID:  data.ProductID,
					SkuName:    skuData.SkuName,
					Price:      skuData.Price,
					TotalStock: skuData.TotalStock,
					Status:     skuData.Status,
				}
				if err := tx.Create(sku).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Model(&domain.ShopSKU{}).
				Where("sku_id = ? AND product_id = ?", skuData.SkuID, data.ProductID).
				Updates(map[string]any{
					"sku_name":    skuData.SkuName,
					"price":       skuData.Price,
					"total_stock": skuData.TotalStock,
					"status":      skuData.Status,
					"version":     gorm.Expr("version + ?", 1),
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ShopRepository) ChangePeripheralStatus(ctx context.Context, productID uint64, status int) (int64, error) {
	var rowsAffected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product domain.ShopProduct
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("product_id = ? AND product_type = ?", productID, domain.ProductTypePeripheral).
			Take(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				rowsAffected = 0
				return nil
			}
			return err
		}

		updates := map[string]any{
			"status": status,
		}
		if product.Status == domain.ProductStatusOnShelf && status == domain.ProductStatusOffShelf {
			updates["last_off_shelf_time"] = time.Now()
		}

		result := tx.Model(&domain.ShopProduct{}).
			Where("product_id = ? AND product_type = ?", productID, domain.ProductTypePeripheral).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected
		if rowsAffected == 0 {
			return nil
		}
		return nil
	})
	return rowsAffected, err
}

func (r *ShopRepository) applyPeripheralListFilter(db *gorm.DB, query PeripheralListQuery) *gorm.DB {
	db = db.Joins(skuSummaryJoinSQL()).
		Where("sp.product_type = ?", domain.ProductTypePeripheral)
	if query.ProductNameFuzzy != "" {
		db = db.Where("sp.product_name LIKE ?", "%"+query.ProductNameFuzzy+"%")
	}
	if query.Status != nil {
		db = db.Where("sp.status = ?", *query.Status)
	}
	if query.SaleStatus != nil {
		switch *query.SaleStatus {
		case domain.SaleStatusPending:
			db = db.Where("sp.status = ? AND sp.sale_start_time IS NOT NULL AND sp.sale_start_time > NOW()", domain.ProductStatusOnShelf)
		case domain.SaleStatusOnSale:
			db = db.Where("sp.status = ? AND (sp.sale_start_time IS NULL OR sp.sale_start_time <= NOW())", domain.ProductStatusOnShelf).
				Where("COALESCE(ss.active_available_stock, 0) > 0")
		case domain.SaleStatusSoldOut:
			db = db.Where("sp.status = ? AND (sp.sale_start_time IS NULL OR sp.sale_start_time <= NOW())", domain.ProductStatusOnShelf).
				Where("COALESCE(ss.active_available_stock, 0) <= 0")
		case domain.SaleStatusOff:
			db = db.Where("sp.status = ?", domain.ProductStatusOffShelf)
		}
	}
	return db
}

func (r *ShopRepository) applyWebPeripheralListFilter(db *gorm.DB, keyword string) *gorm.DB {
	db = db.Joins(skuSummaryJoinSQL()).
		Where("sp.product_type = ?", domain.ProductTypePeripheral).
		Where("sp.status = ?", domain.ProductStatusOnShelf).
		Where("COALESCE(ss.active_sku_count, 0) > 0")
	if keyword != "" {
		db = db.Where("sp.product_name LIKE ?", "%"+keyword+"%")
	}
	return db
}

func adminPeripheralSelectSQL() string {
	return `
		sp.product_id,
		COALESCE(sp.product_name, '') AS product_name,
		COALESCE(sp.cover_url, '') AS cover_url,
		COALESCE(sp.description, '') AS description,
		sp.status,
		sp.recommend_status,
		COALESCE(DATE_FORMAT(sp.sale_start_time, '%Y-%m-%d %H:%i:%s'), '') AS sale_start_time,
		COALESCE(DATE_FORMAT(sp.last_off_shelf_time, '%Y-%m-%d %H:%i:%s'), '') AS last_off_shelf_time,
		sp.sort,
		COALESCE(ss.sku_id, 0) AS sku_id,
		COALESCE(ss.sku_name, '') AS sku_name,
		COALESCE(ss.min_price, 0) AS price,
		COALESCE(ss.max_price, 0) AS max_price,
		COALESCE(ss.total_stock, 0) AS total_stock,
		COALESCE(ss.locked_stock, 0) AS locked_stock,
		COALESCE(ss.sold_stock, 0) AS sold_stock,
		COALESCE(ss.available_stock, 0) AS available_stock,
		CASE
			WHEN sp.status = 0 THEN 3
			WHEN sp.sale_start_time IS NOT NULL AND sp.sale_start_time > NOW() THEN 0
			WHEN COALESCE(ss.active_available_stock, 0) <= 0 THEN 2
			ELSE 1
		END AS sale_status,
		COALESCE(DATE_FORMAT(sp.create_time, '%Y-%m-%d %H:%i:%s'), '') AS create_time,
		COALESCE(DATE_FORMAT(sp.update_time, '%Y-%m-%d %H:%i:%s'), '') AS update_time
	`
}

func webPeripheralSelectSQL() string {
	return `
		sp.product_id AS item_id,
		sp.product_id,
		COALESCE(ss.sku_id, 0) AS sku_id,
		COALESCE(sp.product_name, '') AS item_name,
		COALESCE(sp.cover_url, '') AS cover_url,
		COALESCE(sp.description, '') AS description,
		COALESCE(ss.min_price, 0) AS price,
		COALESCE(ss.max_price, 0) AS max_price,
		COALESCE(ss.total_stock, 0) AS total_stock,
		COALESCE(ss.active_available_stock, 0) AS available_stock,
		COALESCE(DATE_FORMAT(sp.sale_start_time, '%Y-%m-%d %H:%i:%s'), '') AS sale_start_time,
		CASE
			WHEN sp.sale_start_time IS NOT NULL AND sp.sale_start_time > NOW() THEN 0
			WHEN COALESCE(ss.active_available_stock, 0) <= 0 THEN 2
			ELSE 1
		END AS sale_status,
		sp.recommend_status
	`
}

func skuSummaryJoinSQL() string {
	return `
		LEFT JOIN (
			SELECT
				product_id,
				MIN(sku_id) AS sku_id,
				MIN(sku_name) AS sku_name,
				MIN(CASE WHEN status = 1 THEN price END) AS min_price,
				MAX(CASE WHEN status = 1 THEN price END) AS max_price,
				SUM(total_stock) AS total_stock,
				SUM(locked_stock) AS locked_stock,
				SUM(sold_stock) AS sold_stock,
				SUM(GREATEST(total_stock - locked_stock - sold_stock, 0)) AS available_stock,
				SUM(CASE WHEN status = 1 THEN GREATEST(total_stock - locked_stock - sold_stock, 0) ELSE 0 END) AS active_available_stock,
				SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS active_sku_count
			FROM shop_sku
			GROUP BY product_id
		) ss ON sp.product_id = ss.product_id
	`
}

var ErrStockLessThanOccupied = errors.New("total stock is less than locked and sold stock")

var ErrPriceChangeTooEarly = errors.New("price can only be changed after off shelf for 30 minutes")

func isPriceChanged(oldPrice float64, newPrice float64) bool {
	return math.Round(oldPrice*100) != math.Round(newPrice*100)
}

func canChangePrice(product domain.ShopProduct) bool {
	if product.Status == domain.ProductStatusOnShelf {
		return false
	}
	if product.LastOffShelfTime == nil {
		return true
	}
	return time.Since(*product.LastOffShelfTime) >= priceChangeWaitDuration
}
