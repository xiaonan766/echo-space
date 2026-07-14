package web

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/embedding"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/service/gallerysearch"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type GalleryHandler struct {
	galleryService *webservice.GalleryService
}

func (h *GalleryHandler) Search(c *gin.Context) {
	input := gallerysearch.SearchInput{
		SearchType:  formOrQuery(c, "searchType"),
		Keyword:     formOrQuery(c, "keyword"),
		SearchToken: formOrQuery(c, "searchToken"),
		PageNo:      parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		PageSize:    parseWebIntWithDefault(formOrQuery(c, "pageSize"), 15),
	}
	if input.SearchToken == "" && input.SearchType == "image" {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			response.BusinessError(c, "请选择查询图片", nil)
			return
		}
		if fileHeader.Size <= 0 || fileHeader.Size > embedding.MaxQueryImageBytes {
			response.BusinessError(c, "图片大小不能超过 10MB", nil)
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			response.BusinessError(c, "读取查询图片失败", nil)
			return
		}
		defer file.Close()
		input.Image, err = io.ReadAll(io.LimitReader(file, embedding.MaxQueryImageBytes+1))
		if err != nil || len(input.Image) > embedding.MaxQueryImageBytes {
			response.BusinessError(c, "读取查询图片失败", nil)
			return
		}
	}

	result, err := h.galleryService.Search(c.Request.Context(), input)
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		if !errors.Is(err, http.ErrBodyReadAfterClose) {
			log.Printf("web gallery vector search: %v", err)
		}
		response.ServerError(c, nil)
		return
	}
	response.Success(c, result)
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
