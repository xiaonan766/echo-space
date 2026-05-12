package admin

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	adminservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/admin"
)

type InteractHandler struct {
	interactService *adminservice.InteractService
}

func NewInteractHandler(interactService *adminservice.InteractService) *InteractHandler {
	return &InteractHandler{
		interactService: interactService,
	}
}

func (h *InteractHandler) LoadComment(c *gin.Context) {
	result, err := h.interactService.LoadComment(c.Request.Context(), adminservice.InteractListInput{
		PageNo:         parseIntWithDefault(c.PostForm("pageNo"), 1),
		PageSize:       parseIntWithDefault(c.PostForm("pageSize"), 15),
		VideoNameFuzzy: c.PostForm("videoNameFuzzy"),
	})
	if err != nil {
		log.Printf("admin load comment: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *InteractHandler) DeleteComment(c *gin.Context) {
	commentID, ok := parseRequiredIntForm(c, "commentId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	if err := h.interactService.DeleteComment(c.Request.Context(), commentID); err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin delete comment: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}

func (h *InteractHandler) LoadDanmu(c *gin.Context) {
	result, err := h.interactService.LoadDanmu(c.Request.Context(), adminservice.InteractListInput{
		PageNo:         parseIntWithDefault(c.PostForm("pageNo"), 1),
		PageSize:       parseIntWithDefault(c.PostForm("pageSize"), 15),
		VideoNameFuzzy: c.PostForm("videoNameFuzzy"),
	})
	if err != nil {
		log.Printf("admin load danmu: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, result)
}

func (h *InteractHandler) DeleteDanmu(c *gin.Context) {
	danmuID, ok := parseRequiredIntForm(c, "danmuId")
	if !ok {
		response.BusinessError(c, "\u53c2\u6570\u9519\u8bef", nil)
		return
	}

	if err := h.interactService.DeleteDanmu(c.Request.Context(), danmuID); err != nil {
		if businessError, ok := adminservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}

		log.Printf("admin delete danmu: %v", err)
		response.ServerError(c, nil)
		return
	}

	response.Success(c, nil)
}
