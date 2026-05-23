package admin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type ShopService struct {
	shopRepository *repository.ShopRepository
	recommendStore *cache.ShopRecommendStore
	stockStore     *cache.ShopStockStore
}

type PeripheralListInput struct {
	PageNo           int
	PageSize         int
	ProductNameFuzzy string
	Status           *int
	SaleStatus       *int
}

type SavePeripheralInput struct {
	ProductID       uint64
	ProductName     string
	CoverURL        string
	Description     string
	SaleStartTime   string
	Status          int
	RecommendStatus int
	Sort            int
	SkuList         []SavePeripheralSKUInput
}

type SavePeripheralSKUInput struct {
	SkuID      uint64
	SkuName    string
	Price      float64
	TotalStock int
	Status     int
}

const peripheralStockPrewarmWindow = 5 * time.Minute

func NewShopService(shopRepository *repository.ShopRepository, recommendStore *cache.ShopRecommendStore, stockStore *cache.ShopStockStore) *ShopService {
	return &ShopService{
		shopRepository: shopRepository,
		recommendStore: recommendStore,
		stockStore:     stockStore,
	}
}

func (s *ShopService) LoadPeripheral(ctx context.Context, input PeripheralListInput) (domain.PaginationResult[domain.AdminPeripheralItem], error) {
	input = normalizePeripheralListInput(input)
	list, totalCount, err := s.shopRepository.ListPeripheralByPage(ctx, repository.PeripheralListQuery{
		PageNo:           input.PageNo,
		PageSize:         input.PageSize,
		ProductNameFuzzy: input.ProductNameFuzzy,
		Status:           input.Status,
		SaleStatus:       input.SaleStatus,
	})
	if err != nil {
		return domain.PaginationResult[domain.AdminPeripheralItem]{}, err
	}
	fillPeripheralNames(list)
	return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
}

func (s *ShopService) GetPeripheral(ctx context.Context, productID uint64) (*domain.AdminPeripheralItem, error) {
	if productID == 0 {
		return nil, &BusinessError{Info: "参数错误"}
	}

	item, err := s.shopRepository.FindPeripheralDetail(ctx, productID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "周边商品不存在"}
	}
	if err != nil {
		return nil, err
	}
	fillPeripheralName(item)
	return item, nil
}

func (s *ShopService) SavePeripheral(ctx context.Context, input SavePeripheralInput) error {
	input.ProductName = strings.TrimSpace(input.ProductName)
	input.CoverURL = strings.TrimSpace(input.CoverURL)
	input.Description = strings.TrimSpace(input.Description)
	input.SaleStartTime = strings.TrimSpace(input.SaleStartTime)

	if input.ProductName == "" {
		return &BusinessError{Info: "请输入商品名称"}
	}
	if len([]rune(input.ProductName)) > 100 {
		return &BusinessError{Info: "商品名称不能超过100个字"}
	}
	if !isValidCommonStatus(input.Status) || !isValidRecommendStatus(input.RecommendStatus) || input.Sort < 0 {
		return &BusinessError{Info: "参数错误"}
	}

	saleStartTime, err := parseAdminDateTime(input.SaleStartTime)
	if err != nil {
		return &BusinessError{Info: "开售时间格式不正确"}
	}

	skuList, businessError := normalizeSavePeripheralSKUList(input.SkuList)
	if businessError != nil {
		return businessError
	}

	saveResult, err := s.shopRepository.SavePeripheral(ctx, repository.SavePeripheralData{
		ProductID:       input.ProductID,
		ProductName:     input.ProductName,
		CoverURL:        input.CoverURL,
		Description:     input.Description,
		SaleStartTime:   saleStartTime,
		Status:          input.Status,
		RecommendStatus: input.RecommendStatus,
		Sort:            input.Sort,
		SkuList:         skuList,
	})
	if errors.Is(err, repository.ErrStockLessThanOccupied) {
		return &BusinessError{Info: "总库存不能小于已售库存和锁定库存之和"}
	}
	if errors.Is(err, repository.ErrPriceChangeTooEarly) {
		return &BusinessError{Info: "已上架商品需要先下架，且下架满30分钟后才能修改价格、库存或新增规格"}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "周边商品不存在"}
	}
	if err != nil {
		return err
	}

	if saveResult.ProductID > 0 {
		s.deletePeripheralShopCache(ctx, saveResult.ProductID, input.Status == domain.ProductStatusOffShelf)
		if saveResult.ShouldPrewarm {
			s.prewarmPeripheralStockIfReady(ctx, saveResult.ProductID, saleStartTime)
		}
	} else {
		s.deleteHotPeripheralRecommendCache(ctx)
	}
	return nil
}

func (s *ShopService) ChangePeripheralStatus(ctx context.Context, productID uint64, status int) error {
	if productID == 0 || !isValidCommonStatus(status) {
		return &BusinessError{Info: "参数错误"}
	}

	changeResult, err := s.shopRepository.ChangePeripheralStatus(ctx, productID, status)
	if err != nil {
		return err
	}
	if changeResult.RowsAffected == 0 {
		return &BusinessError{Info: "周边商品不存在"}
	}

	s.deletePeripheralShopCache(ctx, productID, status == domain.ProductStatusOffShelf)
	if changeResult.ShouldPrewarm {
		s.prewarmPeripheralStockByProduct(ctx, productID)
	}
	return nil
}

func (s *ShopService) deletePeripheralShopCache(ctx context.Context, productID uint64, deleteHotMarker bool) {
	s.deleteHotPeripheralRecommendCache(ctx)
	if s.recommendStore == nil || productID == 0 {
		return
	}
	if err := s.recommendStore.DeletePeripheralDetail(ctx, productID); err != nil {
		log.Printf("delete peripheral detail cache: productID=%d err=%v", productID, err)
	}
	if deleteHotMarker {
		if err := s.recommendStore.DeleteHotPeripheralMarker(ctx, productID); err != nil {
			log.Printf("delete hot peripheral marker: productID=%d err=%v", productID, err)
		}
	}
}

func (s *ShopService) deleteHotPeripheralRecommendCache(ctx context.Context) {
	if s.recommendStore == nil {
		return
	}
	if err := s.recommendStore.DeleteHotPeripheral(ctx); err != nil {
		log.Printf("delete hot peripheral recommend cache: %v", err)
	}
}

func (s *ShopService) prewarmPeripheralStockIfReady(ctx context.Context, productID uint64, saleStartTime *time.Time) {
	if !shouldPrewarmPeripheralStock(saleStartTime, time.Now()) {
		return
	}
	s.prewarmPeripheralStockByProduct(ctx, productID)
}

func (s *ShopService) prewarmPeripheralStockByProduct(ctx context.Context, productID uint64) {
	if s.stockStore == nil || productID == 0 {
		return
	}

	list, err := s.shopRepository.ListPeripheralSKUStockForPrewarm(ctx, productID)
	if err != nil {
		log.Printf("list peripheral sku stock for prewarm failed: productID=%d err=%v", productID, err)
		return
	}
	if len(list) == 0 || !shouldPrewarmPeripheralStock(list[0].SaleStartTime, time.Now()) {
		return
	}
	for _, item := range list {
		if err := s.stockStore.PrewarmSKUStock(ctx, item.SkuID, item.AvailableStock); err != nil {
			log.Printf("prewarm peripheral sku stock failed: productID=%d skuID=%d err=%v", item.ProductID, item.SkuID, err)
		}
	}
}

func shouldPrewarmPeripheralStock(saleStartTime *time.Time, now time.Time) bool {
	if saleStartTime == nil {
		return true
	}
	return !saleStartTime.After(now.Add(peripheralStockPrewarmWindow))
}

func normalizePeripheralListInput(input PeripheralListInput) PeripheralListInput {
	if input.PageNo <= 0 {
		input.PageNo = defaultPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultPageSize
	}
	if input.PageSize > maxPageSize {
		input.PageSize = maxPageSize
	}
	input.ProductNameFuzzy = strings.TrimSpace(input.ProductNameFuzzy)
	if input.Status != nil && !isValidCommonStatus(*input.Status) {
		input.Status = nil
	}
	if input.SaleStatus != nil && (*input.SaleStatus < domain.SaleStatusPending || *input.SaleStatus > domain.SaleStatusOff) {
		input.SaleStatus = nil
	}
	return input
}

func normalizeSavePeripheralSKUList(input []SavePeripheralSKUInput) ([]repository.SavePeripheralSKUData, *BusinessError) {
	if len(input) == 0 {
		return nil, &BusinessError{Info: "请至少添加一个商品规格"}
	}

	result := make([]repository.SavePeripheralSKUData, 0, len(input))
	seenName := make(map[string]struct{}, len(input))
	activeCount := 0
	for _, sku := range input {
		sku.SkuName = strings.TrimSpace(sku.SkuName)
		if sku.SkuName == "" {
			return nil, &BusinessError{Info: "请输入规格名称"}
		}
		if len([]rune(sku.SkuName)) > 80 {
			return nil, &BusinessError{Info: "规格名称不能超过80个字"}
		}
		if _, ok := seenName[sku.SkuName]; ok {
			return nil, &BusinessError{Info: "规格名称不能重复"}
		}
		seenName[sku.SkuName] = struct{}{}

		if sku.Price <= 0 || math.IsNaN(sku.Price) || math.IsInf(sku.Price, 0) {
			return nil, &BusinessError{Info: "请输入正确的规格单价"}
		}
		if sku.TotalStock < 0 {
			return nil, &BusinessError{Info: "规格库存数量不能小于0"}
		}
		if !isValidCommonStatus(sku.Status) {
			return nil, &BusinessError{Info: "规格状态参数错误"}
		}
		if sku.Status == domain.ProductStatusOnShelf {
			activeCount++
		}

		result = append(result, repository.SavePeripheralSKUData{
			SkuID:      sku.SkuID,
			SkuName:    sku.SkuName,
			Price:      roundMoney(sku.Price),
			TotalStock: sku.TotalStock,
			Status:     sku.Status,
		})
	}
	if activeCount == 0 {
		return nil, &BusinessError{Info: "请至少启用一个商品规格"}
	}
	return result, nil
}

func parseAdminDateTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		time.RFC3339,
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, errors.New("invalid datetime")
}

func fillPeripheralNames(items []domain.AdminPeripheralItem) {
	for index := range items {
		fillPeripheralName(&items[index])
	}
}

func fillPeripheralName(item *domain.AdminPeripheralItem) {
	if item == nil {
		return
	}
	if item.Status == domain.ProductStatusOnShelf {
		item.StatusName = "已上架"
	} else {
		item.StatusName = "已下架"
	}
	switch item.SaleStatus {
	case domain.SaleStatusPending:
		item.SaleStatusName = "待开售"
	case domain.SaleStatusOnSale:
		item.SaleStatusName = "在售"
	case domain.SaleStatusSoldOut:
		item.SaleStatusName = "已售罄"
	default:
		item.SaleStatusName = "已下架"
	}
	if item.SkuName == "" {
		item.SkuName = "默认规格"
	}
	item.PriceText = buildPriceText(item.Price, item.MaxPrice)
	for index := range item.SkuList {
		if item.SkuList[index].SkuName == "" {
			item.SkuList[index].SkuName = "默认规格"
		}
	}
}

func isValidCommonStatus(status int) bool {
	return status == domain.ProductStatusOffShelf || status == domain.ProductStatusOnShelf
}

func isValidRecommendStatus(status int) bool {
	return status == domain.RecommendStatusNo || status == domain.RecommendStatusYes
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func buildPriceText(minPrice float64, maxPrice float64) string {
	if minPrice <= 0 {
		return "价格待定"
	}
	if maxPrice > minPrice {
		return formatMoney(minPrice) + " - " + formatMoney(maxPrice)
	}
	return formatMoney(minPrice)
}

func formatMoney(value float64) string {
	return "¥" + strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}
