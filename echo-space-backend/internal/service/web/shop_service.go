package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	shopItemTypeShow  = "show"
	shopItemTypeGoods = "goods"

	defaultShopPageNo    = 1
	defaultShopPageSize  = 8
	maxShopPageSize      = 50
	defaultRecommendSize = 6
)

type ShopService struct {
	shopRepository *repository.ShopRepository
}

type ShopListInput struct {
	ItemType string
	Keyword  string
	PageNo   int
	PageSize int
}

func NewShopService(shopRepository *repository.ShopRepository) *ShopService {
	return &ShopService{
		shopRepository: shopRepository,
	}
}

func (s *ShopService) LoadRecommend(ctx context.Context, itemType string) ([]domain.WebShopItem, error) {
	itemType = normalizeShopItemType(itemType)
	if itemType == shopItemTypeShow {
		return []domain.WebShopItem{}, nil
	}

	list, err := s.shopRepository.ListRecommendedPeripheralForWeb(ctx, defaultRecommendSize)
	if err != nil {
		return nil, err
	}
	fillWebShopItems(list)
	return list, nil
}

func (s *ShopService) LoadList(ctx context.Context, input ShopListInput) (domain.PaginationResult[domain.WebShopItem], error) {
	input = normalizeShopListInput(input)
	if input.ItemType == shopItemTypeShow {
		return domain.NewPaginationResult([]domain.WebShopItem{}, 0, input.PageNo, input.PageSize), nil
	}

	list, totalCount, err := s.shopRepository.ListPeripheralForWeb(ctx, repository.WebShopListQuery{
		PageNo:   input.PageNo,
		PageSize: input.PageSize,
		Keyword:  input.Keyword,
	})
	if err != nil {
		return domain.PaginationResult[domain.WebShopItem]{}, err
	}
	fillWebShopItems(list)
	return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
}

func normalizeShopListInput(input ShopListInput) ShopListInput {
	input.ItemType = normalizeShopItemType(input.ItemType)
	input.Keyword = strings.TrimSpace(input.Keyword)
	if input.PageNo <= 0 {
		input.PageNo = defaultShopPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultShopPageSize
	}
	if input.PageSize > maxShopPageSize {
		input.PageSize = maxShopPageSize
	}
	return input
}

func normalizeShopItemType(itemType string) string {
	itemType = strings.TrimSpace(itemType)
	if itemType == shopItemTypeGoods {
		return shopItemTypeGoods
	}
	return shopItemTypeShow
}

func fillWebShopItems(items []domain.WebShopItem) {
	for index := range items {
		fillWebShopItem(&items[index])
	}
}

func fillWebShopItem(item *domain.WebShopItem) {
	if item == nil {
		return
	}
	item.PriceText = fmt.Sprintf("￥%.2f", item.Price)
	if item.AvailableStock > 0 {
		item.StockText = fmt.Sprintf("库存 %d", item.AvailableStock)
	} else {
		item.StockText = "暂无库存"
	}
	if item.SaleStartTime != "" {
		item.SaleStartText = "开售时间 " + item.SaleStartTime
	}

	switch item.SaleStatus {
	case domain.SaleStatusPending:
		item.SaleStatusName = "待开售"
	case domain.SaleStatusSoldOut:
		item.SaleStatusName = "已售罄"
	default:
		item.SaleStatusName = "在售"
	}
	item.StatusName = item.SaleStatusName
}
