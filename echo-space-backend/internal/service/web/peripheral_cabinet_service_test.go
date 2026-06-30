package web

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type fakePeripheralCabinetRepository struct {
	visible         bool
	visibleErr      error
	list            []domain.WebPeripheralCabinetItem
	totalCount      int64
	listCalled      bool
	includeHidden   bool
	updateVisible   *bool
	paidSKU         *repository.PaidCabinetSKU
	paidErr         error
	hiddenSkuID     uint64
	hiddenProductID uint64
	shownSkuID      uint64
}

func (r *fakePeripheralCabinetRepository) GetCabinetVisible(ctx context.Context, userID string) (bool, error) {
	return r.visible, r.visibleErr
}

func (r *fakePeripheralCabinetRepository) UpdateCabinetVisible(ctx context.Context, userID string, visible bool) error {
	r.updateVisible = &visible
	return nil
}

func (r *fakePeripheralCabinetRepository) ListPeripheralCabinet(ctx context.Context, query repository.PeripheralCabinetQuery) ([]domain.WebPeripheralCabinetItem, int64, error) {
	r.listCalled = true
	r.includeHidden = query.IncludeHidden
	return r.list, r.totalCount, nil
}

func (r *fakePeripheralCabinetRepository) FindPaidCabinetSKU(ctx context.Context, userID string, skuID uint64) (*repository.PaidCabinetSKU, error) {
	if r.paidErr != nil {
		return nil, r.paidErr
	}
	return r.paidSKU, nil
}

func (r *fakePeripheralCabinetRepository) HideCabinetItem(ctx context.Context, userID string, productID uint64, skuID uint64, hideTime time.Time) error {
	r.hiddenProductID = productID
	r.hiddenSkuID = skuID
	return nil
}

func (r *fakePeripheralCabinetRepository) ShowCabinetItem(ctx context.Context, userID string, skuID uint64) error {
	r.shownSkuID = skuID
	return nil
}

func TestPeripheralCabinetVisitorHiddenSkipsList(t *testing.T) {
	repository := &fakePeripheralCabinetRepository{visible: false}
	service := NewPeripheralCabinetService(repository)

	result, err := service.LoadPeripheralCabinet(context.Background(), "", "1000000001", 1, 12)
	if err != nil {
		t.Fatalf("LoadPeripheralCabinet error = %v", err)
	}
	if result.Owner {
		t.Fatal("Owner = true, want false")
	}
	if result.CabinetVisible {
		t.Fatal("CabinetVisible = true, want false")
	}
	if repository.listCalled {
		t.Fatal("ListPeripheralCabinet called for hidden visitor cabinet")
	}
	if len(result.List) != 0 || result.TotalCount != 0 {
		t.Fatalf("result list/count = %d/%d, want empty", len(result.List), result.TotalCount)
	}
}

func TestPeripheralCabinetOwnerCanLoadHiddenItems(t *testing.T) {
	repository := &fakePeripheralCabinetRepository{
		visible: false,
		list: []domain.WebPeripheralCabinetItem{
			{ProductID: 1, SkuID: 2, ProductName: "键盘", OwnedQuantity: 2, Hidden: true},
		},
		totalCount: 1,
	}
	service := NewPeripheralCabinetService(repository)

	result, err := service.LoadPeripheralCabinet(context.Background(), "1000000001", "1000000001", 1, 12)
	if err != nil {
		t.Fatalf("LoadPeripheralCabinet error = %v", err)
	}
	if !result.Owner {
		t.Fatal("Owner = false, want true")
	}
	if !repository.listCalled || !repository.includeHidden {
		t.Fatalf("listCalled/includeHidden = %v/%v, want true/true", repository.listCalled, repository.includeHidden)
	}
	if len(result.List) != 1 || !result.List[0].Hidden {
		t.Fatalf("result list = %#v, want one hidden item", result.List)
	}
}

func TestPeripheralCabinetUpdateItemRejectsUnpurchasedSKU(t *testing.T) {
	service := NewPeripheralCabinetService(&fakePeripheralCabinetRepository{paidErr: gorm.ErrRecordNotFound})

	err := service.UpdatePeripheralCabinetItemVisible(context.Background(), "1000000001", 2, false)
	if err == nil {
		t.Fatal("UpdatePeripheralCabinetItemVisible error = nil, want business error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error type = %T, want BusinessError", err)
	}
}

func TestPeripheralCabinetUpdateItemHidesPaidSKU(t *testing.T) {
	repository := &fakePeripheralCabinetRepository{paidSKU: &repository.PaidCabinetSKU{ProductID: 9, SkuID: 2}}
	service := NewPeripheralCabinetService(repository)
	service.now = func() time.Time {
		return time.Date(2026, 6, 30, 10, 0, 0, 0, time.Local)
	}

	if err := service.UpdatePeripheralCabinetItemVisible(context.Background(), "1000000001", 2, false); err != nil {
		t.Fatalf("UpdatePeripheralCabinetItemVisible error = %v", err)
	}
	if repository.hiddenProductID != 9 || repository.hiddenSkuID != 2 {
		t.Fatalf("hidden product/sku = %d/%d, want 9/2", repository.hiddenProductID, repository.hiddenSkuID)
	}
}

func TestPeripheralCabinetReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("db error")
	service := NewPeripheralCabinetService(&fakePeripheralCabinetRepository{visibleErr: expectedErr})

	_, err := service.LoadPeripheralCabinet(context.Background(), "", "1000000001", 1, 12)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}
