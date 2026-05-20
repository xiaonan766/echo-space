package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/middleware"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type ShopOrderHandler struct {
	orderService *webservice.ShopOrderService
}

func NewShopOrderHandler(orderService *webservice.ShopOrderService) *ShopOrderHandler {
	return &ShopOrderHandler{
		orderService: orderService,
	}
}

func (h *ShopOrderHandler) CreateOrder(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	productID, ok := parseWebRequiredUint64(formOrQuery(c, "productId"))
	if !ok {
		response.BusinessError(c, "参数错误", nil)
		return
	}
	skuID, ok := parseWebRequiredUint64(formOrQuery(c, "skuId"))
	if !ok {
		response.BusinessError(c, "参数错误", nil)
		return
	}

	result, err := h.orderService.CreateOrder(c.Request.Context(), webservice.CreateShopOrderInput{
		UserID:    tokenUserInfo.UserID,
		ProductID: productID,
		SkuID:     skuID,
		BuyCount:  parseWebIntWithDefault(formOrQuery(c, "buyCount"), 1),
		RequestID: formOrQuery(c, "requestId"),
	})
	if err != nil {
		handleWebOrderError(c, "web create shop order", err)
		return
	}
	response.Success(c, result)
}

func (h *ShopOrderHandler) GetOrderDetail(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	result, err := h.orderService.GetOrderDetail(c.Request.Context(), tokenUserInfo.UserID, formOrQuery(c, "orderNo"))
	if err != nil {
		handleWebOrderError(c, "web get shop order detail", err)
		return
	}
	response.Success(c, result)
}

func (h *ShopOrderHandler) LoadOrder(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	result, err := h.orderService.LoadOrder(
		c.Request.Context(),
		tokenUserInfo.UserID,
		parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		parseWebIntWithDefault(formOrQuery(c, "pageSize"), 10),
	)
	if err != nil {
		handleWebOrderError(c, "web load shop order", err)
		return
	}
	response.Success(c, result)
}

func getWebTokenUserInfo(c *gin.Context) (*webservice.TokenUserInfo, bool) {
	value, exists := c.Get(middleware.ContextWebTokenUserInfoKey)
	if !exists {
		return nil, false
	}
	tokenUserInfo, ok := value.(*webservice.TokenUserInfo)
	if !ok || tokenUserInfo == nil || tokenUserInfo.UserID == "" {
		return nil, false
	}
	return tokenUserInfo, true
}

func handleWebOrderError(c *gin.Context, logPrefix string, err error) {
	if businessError, ok := webservice.IsBusinessError(err); ok {
		response.BusinessError(c, businessError.Info, nil)
		return
	}

	log.Printf("%s: %v", logPrefix, err)
	response.ServerError(c, nil)
}
