package web

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type GalleryHandler struct {
	galleryService *webservice.GalleryService
}

func NewGalleryHandler(galleryService *webservice.GalleryService) *GalleryHandler {
	return &GalleryHandler{galleryService: galleryService}
}

func (h *GalleryHandler) LoadImageList(c *gin.Context) {
	result, err := h.galleryService.LoadImageList(c.Request.Context(), webservice.GalleryImageListInput{
		PageNo:   parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		PageSize: parseWebIntWithDefault(formOrQuery(c, "pageSize"), 15),
	})
	if err != nil {
		log.Printf("web load gallery image list: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *GalleryHandler) GetImageInfo(c *gin.Context) {
	result, err := h.galleryService.GetImageInfo(c.Request.Context(), formOrQuery(c, "imageId"))
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web get gallery image info: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}
