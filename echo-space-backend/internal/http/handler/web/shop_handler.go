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
