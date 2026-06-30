package domain

import "time"

const (
	OrderStatusStockLocking = 0
	OrderStatusWaitPay      = 1
	OrderStatusStockFailed  = 2
	OrderStatusPaid         = 3
	OrderStatusCanceled     = 4
	OrderStatusTimeout      = 5

	PayStatusUnpaid = 0
	PayStatusPaid   = 1

	StockFlowTypeLock   = 1
	StockFlowTypeUnlock = 2
	StockFlowTypeSold   = 3

	ShopCoinFlowTypePay = 1

	OrderMessageTypeStockLock = 1

	OrderMessageStatusWaitPublish     = 0
	OrderMessageStatusPublished       = 1
	OrderMessageStatusConsumedSuccess = 2
	OrderMessageStatusConsumedFailed  = 3
	OrderMessageStatusDead            = 4
)

type ShopOrder struct {
	OrderID         uint64     `gorm:"primaryKey;autoIncrement;column:order_id" json:"orderId"`
	OrderNo         string     `gorm:"column:order_no;type:varchar(32);not null" json:"orderNo"`
	UserID          string     `gorm:"column:user_id;type:varchar(10);not null" json:"userId"`
	OrderStatus     int        `gorm:"column:order_status;type:tinyint;not null;default:0" json:"orderStatus"`
	PayStatus       int        `gorm:"column:pay_status;type:tinyint;not null;default:0" json:"payStatus"`
	TotalAmount     float64    `gorm:"column:total_amount;not null;default:0" json:"totalAmount"`
	PayAmount       float64    `gorm:"column:pay_amount;not null;default:0" json:"payAmount"`
	DeliveryType    int        `gorm:"column:delivery_type;type:tinyint;not null;default:0" json:"deliveryType"`
	ReceiverName    string     `gorm:"column:receiver_name;type:varchar(50)" json:"receiverName"`
	ReceiverPhone   string     `gorm:"column:receiver_phone;type:varchar(20)" json:"receiverPhone"`
	ReceiverAddress string     `gorm:"column:receiver_address;type:varchar(255)" json:"receiverAddress"`
	ExpireTime      time.Time  `gorm:"column:expire_time;not null" json:"expireTime"`
	PayTime         *time.Time `gorm:"column:pay_time" json:"payTime"`
	CancelTime      *time.Time `gorm:"column:cancel_time" json:"cancelTime"`
	CreateTime      time.Time  `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime      time.Time  `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
}

func (ShopOrder) TableName() string {
	return "shop_order"
}

type ShopOrderItem struct {
	ItemID      uint64    `gorm:"primaryKey;autoIncrement;column:item_id" json:"itemId"`
	OrderID     uint64    `gorm:"column:order_id;not null" json:"orderId"`
	OrderNo     string    `gorm:"column:order_no;type:varchar(32);not null" json:"orderNo"`
	ProductID   uint64    `gorm:"column:product_id;not null" json:"productId"`
	SkuID       uint64    `gorm:"column:sku_id;not null" json:"skuId"`
	ProductName string    `gorm:"column:product_name;type:varchar(100);not null" json:"productName"`
	SkuName     string    `gorm:"column:sku_name;type:varchar(80);not null" json:"skuName"`
	CoverURL    string    `gorm:"column:cover_url;type:varchar(255)" json:"coverUrl"`
	Price       float64   `gorm:"column:price;not null" json:"price"`
	Quantity    int       `gorm:"column:quantity;not null" json:"quantity"`
	TotalAmount float64   `gorm:"column:total_amount;not null" json:"totalAmount"`
	CreateTime  time.Time `gorm:"column:create_time;autoCreateTime" json:"createTime"`
}

func (ShopOrderItem) TableName() string {
	return "shop_order_item"
}

type ShopStockFlow struct {
	FlowID            uint64    `gorm:"primaryKey;autoIncrement;column:flow_id" json:"flowId"`
	SkuID             uint64    `gorm:"column:sku_id;not null" json:"skuId"`
	OrderNo           string    `gorm:"column:order_no;type:varchar(32);not null" json:"orderNo"`
	ChangeType        int       `gorm:"column:change_type;type:tinyint;not null" json:"changeType"`
	ChangeCount       int       `gorm:"column:change_count;not null" json:"changeCount"`
	BeforeLockedStock int       `gorm:"column:before_locked_stock" json:"beforeLockedStock"`
	AfterLockedStock  int       `gorm:"column:after_locked_stock" json:"afterLockedStock"`
	BeforeSoldStock   int       `gorm:"column:before_sold_stock" json:"beforeSoldStock"`
	AfterSoldStock    int       `gorm:"column:after_sold_stock" json:"afterSoldStock"`
	CreateTime        time.Time `gorm:"column:create_time;autoCreateTime" json:"createTime"`
}

func (ShopStockFlow) TableName() string {
	return "shop_stock_flow"
}

type ShopCoinFlow struct {
	FlowID     uint64    `gorm:"primaryKey;autoIncrement;column:flow_id" json:"flowId"`
	UserID     string    `gorm:"column:user_id;type:varchar(10);not null" json:"userId"`
	OrderNo    string    `gorm:"column:order_no;type:varchar(32);not null" json:"orderNo"`
	ChangeType int       `gorm:"column:change_type;type:tinyint;not null" json:"changeType"`
	ChangeCoin int       `gorm:"column:change_coin;not null" json:"changeCoin"`
	BeforeCoin int       `gorm:"column:before_coin;not null" json:"beforeCoin"`
	AfterCoin  int       `gorm:"column:after_coin;not null" json:"afterCoin"`
	Remark     string    `gorm:"column:remark;type:varchar(255)" json:"remark"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"createTime"`
}

func (ShopCoinFlow) TableName() string {
	return "shop_coin_flow"
}

type ShopOrderMessage struct {
	MessageID     uint64     `gorm:"primaryKey;autoIncrement;column:message_id" json:"messageId"`
	OrderNo       string     `gorm:"column:order_no;type:varchar(32);not null" json:"orderNo"`
	MessageType   int        `gorm:"column:message_type;type:tinyint;not null" json:"messageType"`
	MessageStatus int        `gorm:"column:message_status;type:tinyint;not null" json:"messageStatus"`
	Payload       string     `gorm:"column:payload;type:json;not null" json:"payload"`
	RetryCount    int        `gorm:"column:retry_count;not null;default:0" json:"retryCount"`
	NextRetryTime *time.Time `gorm:"column:next_retry_time" json:"nextRetryTime"`
	LastError     string     `gorm:"column:last_error;type:varchar(500)" json:"lastError"`
	CreateTime    time.Time  `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime    time.Time  `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
}

func (ShopOrderMessage) TableName() string {
	return "shop_order_message"
}

type WebShopOrderItem struct {
	OrderID          uint64  `gorm:"column:order_id" json:"orderId"`
	OrderNo          string  `gorm:"column:order_no" json:"orderNo"`
	UserID           string  `gorm:"column:user_id" json:"userId"`
	OrderStatus      int     `gorm:"column:order_status" json:"orderStatus"`
	OrderStatusName  string  `gorm:"-" json:"orderStatusName"`
	PayStatus        int     `gorm:"column:pay_status" json:"payStatus"`
	TotalAmount      float64 `gorm:"column:total_amount" json:"totalAmount"`
	TotalAmountText  string  `gorm:"-" json:"totalAmountText"`
	PayAmount        float64 `gorm:"column:pay_amount" json:"payAmount"`
	ProductID        uint64  `gorm:"column:product_id" json:"productId"`
	SkuID            uint64  `gorm:"column:sku_id" json:"skuId"`
	ProductName      string  `gorm:"column:product_name" json:"productName"`
	SkuName          string  `gorm:"column:sku_name" json:"skuName"`
	CoverURL         string  `gorm:"column:cover_url" json:"coverUrl"`
	Price            float64 `gorm:"column:price" json:"price"`
	PriceText        string  `gorm:"-" json:"priceText"`
	Quantity         int     `gorm:"column:quantity" json:"quantity"`
	CurrentCoinCount int     `gorm:"-" json:"currentCoinCount"`
	ExpireTime       string  `gorm:"column:expire_time" json:"expireTime"`
	CreateTime       string  `gorm:"column:create_time" json:"createTime"`
	UpdateTime       string  `gorm:"column:update_time" json:"updateTime"`
}
