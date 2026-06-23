package web

import (
	"context"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

func TestSysSettingServiceReturnsDefaultWhenStoreMissing(t *testing.T) {
	service := NewSysSettingService(nil)
	setting, err := service.GetSetting(context.Background())
	if err != nil {
		t.Fatalf("GetSetting error = %v", err)
	}
	if setting != domain.DefaultSysSetting() {
		t.Fatalf("setting = %+v, want %+v", setting, domain.DefaultSysSetting())
	}
}
