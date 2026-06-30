package web

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	defaultPeripheralCabinetPageNo   = 1
	defaultPeripheralCabinetPageSize = 12
	maxPeripheralCabinetPageSize     = 50
)

type PeripheralCabinetRepository interface {
	GetCabinetVisible(ctx context.Context, userID string) (bool, error)
	UpdateCabinetVisible(ctx context.Context, userID string, visible bool) error
	ListPeripheralCabinet(ctx context.Context, query repository.PeripheralCabinetQuery) ([]domain.WebPeripheralCabinetItem, int64, error)
	FindPaidCabinetSKU(ctx context.Context, userID string, skuID uint64) (*repository.PaidCabinetSKU, error)
	HideCabinetItem(ctx context.Context, userID string, productID uint64, skuID uint64, hideTime time.Time) error
	ShowCabinetItem(ctx context.Context, userID string, skuID uint64) error
}

type PeripheralCabinetService struct {
	repository PeripheralCabinetRepository
	now        func() time.Time
}

func NewPeripheralCabinetService(repository PeripheralCabinetRepository) *PeripheralCabinetService {
	return &PeripheralCabinetService{
		repository: repository,
		now:        time.Now,
	}
}

func (s *PeripheralCabinetService) LoadPeripheralCabinet(ctx context.Context, currentUserID string, targetUserID string, pageNo int, pageSize int) (domain.PeripheralCabinetResult, error) {
	currentUserID = strings.TrimSpace(currentUserID)
	targetUserID = strings.TrimSpace(targetUserID)
	if !validWebUserID(targetUserID) || (currentUserID != "" && !validWebUserID(currentUserID)) {
		return domain.PeripheralCabinetResult{}, &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return domain.PeripheralCabinetResult{}, errors.New("peripheral cabinet service is not ready")
	}

	pageNo, pageSize = normalizePeripheralCabinetPage(pageNo, pageSize)
	owner := currentUserID != "" && currentUserID == targetUserID
	cabinetVisible, err := s.repository.GetCabinetVisible(ctx, targetUserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.PeripheralCabinetResult{}, &NotFoundError{Info: "用户不存在"}
	}
	if err != nil {
		return domain.PeripheralCabinetResult{}, err
	}
	if !owner && !cabinetVisible {
		return domain.NewPeripheralCabinetResult(owner, false, []domain.WebPeripheralCabinetItem{}, 0, pageNo, pageSize), nil
	}

	list, totalCount, err := s.repository.ListPeripheralCabinet(ctx, repository.PeripheralCabinetQuery{
		UserID:        targetUserID,
		PageNo:        pageNo,
		PageSize:      pageSize,
		IncludeHidden: owner,
	})
	if err != nil {
		return domain.PeripheralCabinetResult{}, err
	}
	return domain.NewPeripheralCabinetResult(owner, cabinetVisible, list, totalCount, pageNo, pageSize), nil
}

func (s *PeripheralCabinetService) UpdatePeripheralCabinetVisible(ctx context.Context, userID string, visible bool) error {
	userID = strings.TrimSpace(userID)
	if !validWebUserID(userID) {
		return &BusinessError{Info: "请先登录"}
	}
	if s == nil || s.repository == nil {
		return errors.New("peripheral cabinet service is not ready")
	}
	return s.repository.UpdateCabinetVisible(ctx, userID, visible)
}

func (s *PeripheralCabinetService) UpdatePeripheralCabinetItemVisible(ctx context.Context, userID string, skuID uint64, visible bool) error {
	userID = strings.TrimSpace(userID)
	if !validWebUserID(userID) || skuID == 0 {
		return &BusinessError{Info: "参数错误"}
	}
	if s == nil || s.repository == nil {
		return errors.New("peripheral cabinet service is not ready")
	}

	paidSKU, err := s.repository.FindPaidCabinetSKU(ctx, userID, skuID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "未购买该周边"}
	}
	if err != nil {
		return err
	}
	if paidSKU == nil {
		return &BusinessError{Info: "未购买该周边"}
	}

	if visible {
		return s.repository.ShowCabinetItem(ctx, userID, skuID)
	}
	return s.repository.HideCabinetItem(ctx, userID, paidSKU.ProductID, skuID, s.currentTime())
}

func (s *PeripheralCabinetService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func normalizePeripheralCabinetPage(pageNo int, pageSize int) (int, int) {
	if pageNo <= 0 {
		pageNo = defaultPeripheralCabinetPageNo
	}
	if pageSize <= 0 {
		pageSize = defaultPeripheralCabinetPageSize
	}
	if pageSize > maxPeripheralCabinetPageSize {
		pageSize = maxPeripheralCabinetPageSize
	}
	return pageNo, pageSize
}
