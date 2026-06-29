package web

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type ShopHandler struct {
	shopService *webservice.ShopService
}

func NewShopHandler(shopService *webservice.ShopService) *ShopHandler {
	return &ShopHandler{
		shopService: shopService,
	}
}

func (h *ShopHandler) LoadRecommend(c *gin.Context) {
	result, err := h.shopService.LoadRecommend(c.Request.Context(), formOrQuery(c, "itemType"))
	if err != nil {
		log.Printf("web load shop recommend: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *ShopHandler) LoadList(c *gin.Context) {
	result, err := h.shopService.LoadList(c.Request.Context(), webservice.ShopListInput{
		ItemType: formOrQuery(c, "itemType"),
		Keyword:  formOrQuery(c, "keyword"),
		PageNo:   parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		PageSize: parseWebIntWithDefault(formOrQuery(c, "pageSize"), 8),
	})
	if err != nil {
		log.Printf("web load shop list: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *ShopHandler) GetPeripheralDetail(c *gin.Context) {
	productID, ok := parseWebRequiredUint64(formOrQuery(c, "productId"))
	if !ok {
		response.BusinessError(c, "参数错误", nil)
		return
	}

	result, err := h.shopService.GetPeripheralDetail(c.Request.Context(), productID)
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("web get peripheral detail: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func formOrQuery(c *gin.Context, key string) string {
	if value := strings.TrimSpace(c.PostForm(key)); value != "" {
		return value
	}
	return strings.TrimSpace(c.Query(key))
}

func parseWebIntWithDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseOptionalWebInt(value string) *int {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseWebRequiredUint64(value string) (uint64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, err == nil
}
