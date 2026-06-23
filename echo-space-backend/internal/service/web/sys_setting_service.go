package web

import (
	"context"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

type SysSettingService struct {
	settingStore *cache.SysSettingStore
}

func NewSysSettingService(settingStore *cache.SysSettingStore) *SysSettingService {
	return &SysSettingService{settingStore: settingStore}
}

func (s *SysSettingService) GetSetting(ctx context.Context) (domain.SysSetting, error) {
	if s == nil || s.settingStore == nil {
		return domain.DefaultSysSetting(), nil
	}

	setting, exists, err := s.settingStore.Get(ctx)
	if err != nil {
		return domain.SysSetting{}, err
	}
	if !exists {
		return domain.DefaultSysSetting(), nil
	}
	return setting, nil
}
