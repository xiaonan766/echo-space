package admin

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

type ShopHandler struct {
	shopService *adminservice.ShopService
}

func NewShopHandler(shopService *adminservice.ShopService) *ShopHandler {
	return &ShopHandler{
		shopService: shopService,
	}
}

func (h *ShopHandler) LoadPeripheral(c *gin.Context) {
	status, ok := parseOptionalStatus(c.PostForm("status"))
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	saleStatus, ok := parseOptionalSaleStatus(c.PostForm("saleStatus"))
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	result, err := h.shopService.LoadPeripheral(c.Request.Context(), adminservice.PeripheralListInput{
		PageNo:           parseIntWithDefault(c.PostForm("pageNo"), 1),
		PageSize:         parseIntWithDefault(c.PostForm("pageSize"), 15),
		ProductNameFuzzy: c.PostForm("productNameFuzzy"),
		Status:           status,
		SaleStatus:       saleStatus,
	})
	if err != nil {
		log.Printf("admin load peripheral: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *ShopHandler) GetPeripheral(c *gin.Context) {
	productID, ok := parseRequiredUint64Form(c, "productId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	result, err := h.shopService.GetPeripheral(c.Request.Context(), productID)
	if err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin get peripheral: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *ShopHandler) SavePeripheral(c *gin.Context) {
	productID, ok := parseOptionalUint64Form(c, "productId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	skuList, ok := parsePeripheralSKUList(c.PostForm("skuList"))
	if !ok {
		response.BusinessError(c, "\u8bf7\u8f93\u5165\u6b63\u786e\u7684\u89c4\u683c\u4fe1\u606f", nil)
		return
	}

	status, ok := parseRequiredStatus(c.PostForm("status"))
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	recommendStatus, ok := parseRequiredStatus(c.PostForm("recommendStatus"))
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	err := h.shopService.SavePeripheral(c.Request.Context(), adminservice.SavePeripheralInput{
		ProductID:       productID,
		ProductName:     c.PostForm("productName"),
		CoverURL:        c.PostForm("coverUrl"),
		Description:     c.PostForm("description"),
		SaleStartTime:   c.PostForm("saleStartTime"),
		Status:          status,
		RecommendStatus: recommendStatus,
		Sort:            parseIntWithDefault(c.PostForm("sort"), 0),
		SkuList:         skuList,
	})
	if err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin save peripheral: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func parsePeripheralSKUList(value string) ([]adminservice.SavePeripheralSKUInput, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}

	var skuList []adminservice.SavePeripheralSKUInput
	if err := json.Unmarshal([]byte(value), &skuList); err != nil {
		return nil, false
	}
	return skuList, true
}

func (h *ShopHandler) ChangePeripheralStatus(c *gin.Context) {
	productID, ok := parseRequiredUint64Form(c, "productId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	status, ok := parseRequiredStatus(c.PostForm("status"))
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	if err := h.shopService.ChangePeripheralStatus(c.Request.Context(), productID, status); err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin change peripheral status: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func parseOptionalSaleStatus(value string) (*int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}

	status, err := strconv.Atoi(value)
	if err != nil {
		return nil, false
	}
	if status < domain.SaleStatusPending || status > domain.SaleStatusOff {
		return nil, false
	}
	return &status, true
}

func parseRequiredUint64Form(c *gin.Context, key string) (uint64, bool) {
	value := strings.TrimSpace(c.PostForm(key))
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, err == nil
}

func parseOptionalUint64Form(c *gin.Context, key string) (uint64, bool) {
	value := strings.TrimSpace(c.PostForm(key))
	if value == "" {
		return 0, true
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, err == nil
}

func parseRequiredFloatForm(c *gin.Context, key string) (float64, bool) {
	value := strings.TrimSpace(c.PostForm(key))
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.ParseFloat(value, 64)
	return parsed, err == nil
}
