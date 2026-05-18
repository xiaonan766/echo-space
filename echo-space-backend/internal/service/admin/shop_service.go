package admin

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type ShopService struct {
	shopRepository *repository.ShopRepository
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
	Price           float64
	TotalStock      int
	SaleStartTime   string
	Status          int
	RecommendStatus int
	Sort            int
}

func NewShopService(shopRepository *repository.ShopRepository) *ShopService {
	return &ShopService{
		shopRepository: shopRepository,
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
		return nil, &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	item, err := s.shopRepository.FindPeripheralDetail(ctx, productID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "\u5468\u8fb9\u5546\u54c1\u4e0d\u5b58\u5728"}
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
		return &BusinessError{Info: "\u8bf7\u8f93\u5165\u5546\u54c1\u540d\u79f0"}
	}
	if len([]rune(input.ProductName)) > 100 {
		return &BusinessError{Info: "\u5546\u54c1\u540d\u79f0\u4e0d\u80fd\u8d85\u8fc7100\u4e2a\u5b57"}
	}
	if input.Price <= 0 || math.IsNaN(input.Price) || math.IsInf(input.Price, 0) {
		return &BusinessError{Info: "\u8bf7\u8f93\u5165\u6b63\u786e\u7684\u5355\u4ef7"}
	}
	if input.TotalStock < 0 {
		return &BusinessError{Info: "\u5e93\u5b58\u6570\u91cf\u4e0d\u80fd\u5c0f\u4e8e0"}
	}
	if !isValidCommonStatus(input.Status) || !isValidRecommendStatus(input.RecommendStatus) || input.Sort < 0 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	saleStartTime, err := parseAdminDateTime(input.SaleStartTime)
	if err != nil {
		return &BusinessError{Info: "\u5f00\u552e\u65f6\u95f4\u683c\u5f0f\u4e0d\u6b63\u786e"}
	}

	err = s.shopRepository.SavePeripheral(ctx, repository.SavePeripheralData{
		ProductID:       input.ProductID,
		ProductName:     input.ProductName,
		CoverURL:        input.CoverURL,
		Description:     input.Description,
		Price:           roundMoney(input.Price),
		TotalStock:      input.TotalStock,
		SaleStartTime:   saleStartTime,
		Status:          input.Status,
		RecommendStatus: input.RecommendStatus,
		Sort:            input.Sort,
	})
	if errors.Is(err, repository.ErrStockLessThanOccupied) {
		return &BusinessError{Info: "\u603b\u5e93\u5b58\u4e0d\u80fd\u5c0f\u4e8e\u5df2\u552e\u5e93\u5b58\u548c\u9501\u5b9a\u5e93\u5b58\u4e4b\u548c"}
	}
	if errors.Is(err, repository.ErrPriceChangeTooEarly) {
		return &BusinessError{Info: "\u5df2\u4e0a\u67b6\u5546\u54c1\u9700\u5148\u4e0b\u67b6\uff0c\u4e14\u4e0b\u67b6\u6ee130\u5206\u949f\u540e\u624d\u80fd\u4fee\u6539\u4ef7\u683c"}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "\u5468\u8fb9\u5546\u54c1\u4e0d\u5b58\u5728"}
	}
	return err
}

func (s *ShopService) ChangePeripheralStatus(ctx context.Context, productID uint64, status int) error {
	if productID == 0 || !isValidCommonStatus(status) {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	rowsAffected, err := s.shopRepository.ChangePeripheralStatus(ctx, productID, status)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return &BusinessError{Info: "\u5468\u8fb9\u5546\u54c1\u4e0d\u5b58\u5728"}
	}
	return nil
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
		item.StatusName = "\u5df2\u4e0a\u67b6"
	} else {
		item.StatusName = "\u5df2\u4e0b\u67b6"
	}
	switch item.SaleStatus {
	case domain.SaleStatusPending:
		item.SaleStatusName = "\u5f85\u5f00\u552e"
	case domain.SaleStatusOnSale:
		item.SaleStatusName = "\u5728\u552e"
	case domain.SaleStatusSoldOut:
		item.SaleStatusName = "\u5df2\u552e\u7f44"
	default:
		item.SaleStatusName = "\u5df2\u4e0b\u67b6"
	}
	if item.SkuName == "" {
		item.SkuName = "默认规格"
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
