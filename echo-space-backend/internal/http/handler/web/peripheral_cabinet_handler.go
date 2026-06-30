package web

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type PeripheralCabinetHandler struct {
	service        *webservice.PeripheralCabinetService
	accountService *webservice.AccountService
}

func NewPeripheralCabinetHandler(service *webservice.PeripheralCabinetService, accountService *webservice.AccountService) *PeripheralCabinetHandler {
	return &PeripheralCabinetHandler{
		service:        service,
		accountService: accountService,
	}
}

func (h *PeripheralCabinetHandler) LoadPeripheralCabinet(c *gin.Context) {
	currentUserID, err := h.optionalUserID(c)
	if err != nil {
		log.Printf("web load peripheral cabinet token: %v", err)
		response.ServerError(c, nil)
		return
	}

	result, err := h.service.LoadPeripheralCabinet(
		c.Request.Context(),
		currentUserID,
		formOrQuery(c, "userId"),
		parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		parseWebIntWithDefault(formOrQuery(c, "pageSize"), 12),
	)
	if err != nil {
		handlePeripheralCabinetError(c, "web load peripheral cabinet", err)
		return
	}
	response.Success(c, result)
}

func (h *PeripheralCabinetHandler) UpdatePeripheralCabinetVisible(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	visible, ok := parseCabinetVisible(formOrQuery(c, "visible"))
	if !ok {
		response.BusinessError(c, "参数错误", nil)
		return
	}
	if err := h.service.UpdatePeripheralCabinetVisible(c.Request.Context(), tokenUserInfo.UserID, visible); err != nil {
		handlePeripheralCabinetError(c, "web update peripheral cabinet visible", err)
		return
	}
	response.Success(c, nil)
}

func (h *PeripheralCabinetHandler) UpdatePeripheralCabinetItemVisible(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}

	skuID, skuOK := parseWebRequiredUint64(formOrQuery(c, "skuId"))
	visible, visibleOK := parseCabinetVisible(formOrQuery(c, "visible"))
	if !skuOK || !visibleOK {
		response.BusinessError(c, "参数错误", nil)
		return
	}
	if err := h.service.UpdatePeripheralCabinetItemVisible(c.Request.Context(), tokenUserInfo.UserID, skuID, visible); err != nil {
		handlePeripheralCabinetError(c, "web update peripheral cabinet item visible", err)
		return
	}
	response.Success(c, nil)
}

func (h *PeripheralCabinetHandler) optionalUserID(c *gin.Context) (string, error) {
	if h == nil || h.accountService == nil {
		return "", nil
	}

	token := getOptionalWebToken(c)
	if token == "" {
		return "", nil
	}

	tokenUserInfo, ok, err := h.accountService.GetTokenUserInfo(c.Request.Context(), token)
	if err != nil {
		return "", err
	}
	if !ok || tokenUserInfo == nil {
		return "", nil
	}
	return tokenUserInfo.UserID, nil
}

func parseCabinetVisible(value string) (bool, bool) {
	value = strings.TrimSpace(value)
	if value == "1" || strings.EqualFold(value, "true") {
		return true, true
	}
	if value == "0" || strings.EqualFold(value, "false") {
		return false, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return false, false
	}
	if parsed == 1 {
		return true, true
	}
	if parsed == 0 {
		return false, true
	}
	return false, false
}

func handlePeripheralCabinetError(c *gin.Context, logPrefix string, err error) {
	if businessError, ok := webservice.IsBusinessError(err); ok {
		response.BusinessError(c, businessError.Info, nil)
		return
	}
	if notFoundError, ok := webservice.IsNotFoundError(err); ok {
		response.Error(c, response.CodeNotFound, notFoundError.Info, nil)
		return
	}
	log.Printf("%s: %v", logPrefix, err)
	response.ServerError(c, nil)
}
