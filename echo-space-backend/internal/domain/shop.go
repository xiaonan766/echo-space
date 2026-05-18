package domain

import "time"

const (
	ProductTypePeripheral = 2

	ProductStatusOffShelf = 0
	ProductStatusOnShelf  = 1

	RecommendStatusNo  = 0
	RecommendStatusYes = 1

	SaleStatusPending = 0
	SaleStatusOnSale  = 1
	SaleStatusSoldOut = 2
	SaleStatusOff     = 3
)

type ShopProduct struct {
	ProductID        uint64     `gorm:"primaryKey;autoIncrement;column:product_id" json:"productId"`
	ProductType      int        `gorm:"column:product_type;type:tinyint;not null" json:"productType"`
	ProductName      string     `gorm:"column:product_name;type:varchar(100);not null" json:"productName"`
	CoverURL         string     `gorm:"column:cover_url;type:varchar(255)" json:"coverUrl"`
	Description      string     `gorm:"column:description" json:"description"`
	Status           int        `gorm:"column:status;type:tinyint;not null;default:0" json:"status"`
	RecommendStatus  int        `gorm:"column:recommend_status;type:tinyint;not null;default:0" json:"recommendStatus"`
	SaleStartTime    *time.Time `gorm:"column:sale_start_time" json:"saleStartTime"`
	LastOffShelfTime *time.Time `gorm:"column:last_off_shelf_time" json:"lastOffShelfTime"`
	Sort             int        `gorm:"column:sort;not null;default:0" json:"sort"`
	CreateTime       time.Time  `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime       time.Time  `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
}

func (ShopProduct) TableName() string {
	return "shop_product"
}

type ShopSKU struct {
	SkuID        uint64    `gorm:"primaryKey;autoIncrement;column:sku_id" json:"skuId"`
	ProductID    uint64    `gorm:"column:product_id;not null" json:"productId"`
	SkuName      string    `gorm:"column:sku_name;type:varchar(80);not null" json:"skuName"`
	Price        float64   `gorm:"column:price;not null" json:"price"`
	TotalStock   int       `gorm:"column:total_stock;not null;default:0" json:"totalStock"`
	LockedStock  int       `gorm:"column:locked_stock;not null;default:0" json:"lockedStock"`
	SoldStock    int       `gorm:"column:sold_stock;not null;default:0" json:"soldStock"`
	LimitPerUser int       `gorm:"column:limit_per_user;not null;default:0" json:"limitPerUser"`
	Status       int       `gorm:"column:status;type:tinyint;not null;default:1" json:"status"`
	Version      int       `gorm:"column:version;not null;default:0" json:"version"`
	CreateTime   time.Time `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime   time.Time `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
}

func (ShopSKU) TableName() string {
	return "shop_sku"
}

type AdminPeripheralItem struct {
	ProductID        uint64               `gorm:"column:product_id" json:"productId"`
	ProductName      string               `gorm:"column:product_name" json:"productName"`
	CoverURL         string               `gorm:"column:cover_url" json:"coverUrl"`
	Description      string               `gorm:"column:description" json:"description"`
	Status           int                  `gorm:"column:status" json:"status"`
	StatusName       string               `gorm:"-" json:"statusName"`
	RecommendStatus  int                  `gorm:"column:recommend_status" json:"recommendStatus"`
	SaleStartTime    string               `gorm:"column:sale_start_time" json:"saleStartTime"`
	LastOffShelfTime string               `gorm:"column:last_off_shelf_time" json:"lastOffShelfTime"`
	Sort             int                  `gorm:"column:sort" json:"sort"`
	SkuID            uint64               `gorm:"column:sku_id" json:"skuId"`
	SkuName          string               `gorm:"column:sku_name" json:"skuName"`
	Price            float64              `gorm:"column:price" json:"price"`
	MaxPrice         float64              `gorm:"column:max_price" json:"maxPrice"`
	PriceText        string               `gorm:"-" json:"priceText"`
	TotalStock       int                  `gorm:"column:total_stock" json:"totalStock"`
	LockedStock      int                  `gorm:"column:locked_stock" json:"lockedStock"`
	SoldStock        int                  `gorm:"column:sold_stock" json:"soldStock"`
	AvailableStock   int                  `gorm:"column:available_stock" json:"availableStock"`
	SaleStatus       int                  `gorm:"column:sale_status" json:"saleStatus"`
	SaleStatusName   string               `gorm:"-" json:"saleStatusName"`
	CreateTime       string               `gorm:"column:create_time" json:"createTime"`
	UpdateTime       string               `gorm:"column:update_time" json:"updateTime"`
	SkuList          []AdminPeripheralSKU `gorm:"-" json:"skuList"`
}

type AdminPeripheralSKU struct {
	SkuID          uint64  `gorm:"column:sku_id" json:"skuId"`
	ProductID      uint64  `gorm:"column:product_id" json:"productId"`
	SkuName        string  `gorm:"column:sku_name" json:"skuName"`
	Price          float64 `gorm:"column:price" json:"price"`
	TotalStock     int     `gorm:"column:total_stock" json:"totalStock"`
	LockedStock    int     `gorm:"column:locked_stock" json:"lockedStock"`
	SoldStock      int     `gorm:"column:sold_stock" json:"soldStock"`
	AvailableStock int     `gorm:"column:available_stock" json:"availableStock"`
	Status         int     `gorm:"column:status" json:"status"`
}

type WebShopItem struct {
	ItemID          uint64       `gorm:"column:item_id" json:"itemId"`
	ProductID       uint64       `gorm:"column:product_id" json:"productId"`
	SkuID           uint64       `gorm:"column:sku_id" json:"skuId"`
	ItemName        string       `gorm:"column:item_name" json:"itemName"`
	CoverURL        string       `gorm:"column:cover_url" json:"coverUrl"`
	Description     string       `gorm:"column:description" json:"description"`
	Price           float64      `gorm:"column:price" json:"price"`
	MaxPrice        float64      `gorm:"column:max_price" json:"maxPrice"`
	PriceText       string       `gorm:"-" json:"priceText"`
	TotalStock      int          `gorm:"column:total_stock" json:"totalStock"`
	AvailableStock  int          `gorm:"column:available_stock" json:"availableStock"`
	StockText       string       `gorm:"-" json:"stockText"`
	SaleStartTime   string       `gorm:"column:sale_start_time" json:"saleStartTime"`
	SaleStartText   string       `gorm:"-" json:"saleStartText"`
	SaleStatus      int          `gorm:"column:sale_status" json:"saleStatus"`
	SaleStatusName  string       `gorm:"-" json:"saleStatusName"`
	StatusName      string       `gorm:"-" json:"statusName"`
	RecommendStatus int          `gorm:"column:recommend_status" json:"recommendStatus"`
	SkuList         []WebShopSKU `gorm:"-" json:"skuList"`
}

type WebShopSKU struct {
	SkuID          uint64  `gorm:"column:sku_id" json:"skuId"`
	SkuName        string  `gorm:"column:sku_name" json:"skuName"`
	Price          float64 `gorm:"column:price" json:"price"`
	PriceText      string  `gorm:"-" json:"priceText"`
	TotalStock     int     `gorm:"column:total_stock" json:"totalStock"`
	AvailableStock int     `gorm:"column:available_stock" json:"availableStock"`
	StockText      string  `gorm:"-" json:"stockText"`
	Status         int     `gorm:"column:status" json:"status"`
	SaleStatus     int     `gorm:"column:sale_status" json:"saleStatus"`
	SaleStatusName string  `gorm:"-" json:"saleStatusName"`
}
