package domain

import "time"

type ShopCabinetHiddenItem struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID    string    `gorm:"column:user_id;type:varchar(10);not null" json:"userId"`
	ProductID uint64    `gorm:"column:product_id;not null;default:0" json:"productId"`
	SkuID     uint64    `gorm:"column:sku_id;not null" json:"skuId"`
	HideTime  time.Time `gorm:"column:hide_time;autoCreateTime" json:"hideTime"`
}

func (ShopCabinetHiddenItem) TableName() string {
	return "shop_cabinet_hidden_item"
}

type WebPeripheralCabinetItem struct {
	ProductID     uint64 `gorm:"column:product_id" json:"productId"`
	SkuID         uint64 `gorm:"column:sku_id" json:"skuId"`
	ProductName   string `gorm:"column:product_name" json:"productName"`
	SkuName       string `gorm:"column:sku_name" json:"skuName"`
	CoverURL      string `gorm:"column:cover_url" json:"coverUrl"`
	OwnedQuantity int    `gorm:"column:owned_quantity" json:"ownedQuantity"`
	OrderCount    int    `gorm:"column:order_count" json:"orderCount"`
	LatestBuyTime string `gorm:"column:latest_buy_time" json:"latestBuyTime"`
	Hidden        bool   `gorm:"column:hidden" json:"hidden"`
}

type PeripheralCabinetResult struct {
	Owner          bool                       `json:"owner"`
	CabinetVisible bool                       `json:"cabinetVisible"`
	TotalCount     int64                      `json:"totalCount"`
	PageSize       int                        `json:"pageSize"`
	PageNo         int                        `json:"pageNo"`
	PageTotal      int                        `json:"pageTotal"`
	List           []WebPeripheralCabinetItem `json:"list"`
}

func NewPeripheralCabinetResult(owner bool, cabinetVisible bool, list []WebPeripheralCabinetItem, totalCount int64, pageNo int, pageSize int) PeripheralCabinetResult {
	if list == nil {
		list = []WebPeripheralCabinetItem{}
	}
	pageTotal := 0
	if pageSize > 0 {
		pageTotal = int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	}
	return PeripheralCabinetResult{
		Owner:          owner,
		CabinetVisible: cabinetVisible,
		TotalCount:     totalCount,
		PageSize:       pageSize,
		PageNo:         pageNo,
		PageTotal:      pageTotal,
		List:           list,
	}
}
