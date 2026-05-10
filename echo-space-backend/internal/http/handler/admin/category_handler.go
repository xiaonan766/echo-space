package admin

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

type CategoryHandler struct {
	categoryService *adminservice.CategoryService
}

func NewCategoryHandler(categoryService *adminservice.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

func (h *CategoryHandler) LoadCategory(c *gin.Context) {
	result, err := h.categoryService.LoadCategory(c.Request.Context())
	if err != nil {
		log.Printf("admin load category: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *CategoryHandler) SaveCategory(c *gin.Context) {
	pCategoryID, ok := parseRequiredIntForm(c, "pCategoryId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	categoryID, ok := parseOptionalIntForm(c, "categoryId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	err := h.categoryService.SaveCategory(c.Request.Context(), adminservice.SaveCategoryInput{
		PCategoryID:  pCategoryID,
		CategoryID:   categoryID,
		CategoryCode: c.PostForm("categoryCode"),
		CategoryName: c.PostForm("categoryName"),
		Icon:         c.PostForm("icon"),
		Background:   c.PostForm("background"),
	})
	if err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin save category: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	categoryID, ok := parseRequiredIntForm(c, "categoryId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	err := h.categoryService.DeleteCategory(c.Request.Context(), categoryID)
	if err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin delete category: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func (h *CategoryHandler) ChangeSort(c *gin.Context) {
	pCategoryID, ok := parseRequiredIntForm(c, "pCategoryId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	err := h.categoryService.ChangeSort(c.Request.Context(), adminservice.ChangeCategorySortInput{
		PCategoryID: pCategoryID,
		CategoryIDs: c.PostForm("categoryIds"),
	})
	if err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin change category sort: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func parseRequiredIntForm(c *gin.Context, key string) (int, bool) {
	value := strings.TrimSpace(c.PostForm(key))
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}

func parseOptionalIntForm(c *gin.Context, key string) (int, bool) {
	value := strings.TrimSpace(c.PostForm(key))
	if value == "" {
		return 0, true
	}

	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}
