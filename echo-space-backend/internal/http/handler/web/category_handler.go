package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type CategoryHandler struct {
	categoryService *webservice.CategoryService
}

func NewCategoryHandler(categoryService *webservice.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

func (h *CategoryHandler) LoadAllCategory(c *gin.Context) {
	result, err := h.categoryService.LoadAllCategory(c.Request.Context())
	if err != nil {
		log.Printf("web load all category: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}
