package admin

import (
	"context"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

const sysSettingKey = "echo-space:sys_setting"

type SettingService struct {
	settingStore *cache.SysSettingStore
}

func NewSettingService(hybridCache *cache.HybridCache) *SettingService {
	return &SettingService{
		settingStore: cache.NewSysSettingStore(hybridCache, sysSettingKey),
	}
}

func (s *SettingService) GetSetting(ctx context.Context) (domain.SysSetting, error) {
	setting, exists, err := s.settingStore.Get(ctx)
	if err != nil {
		return domain.SysSetting{}, err
	}
	if !exists {
		return domain.DefaultSysSetting(), nil
	}
	return domain.NormalizeSysSetting(setting), nil
}

func (s *SettingService) SaveSetting(ctx context.Context, setting domain.SysSetting) error {
	setting = domain.NormalizeSysSetting(setting)
	if !validSysSetting(setting) {
		return &BusinessError{Info: "参数错误"}
	}
	return s.settingStore.Save(ctx, setting)
}

func validSysSetting(setting domain.SysSetting) bool {
	return setting.RegisterCoinCount > 0 &&
		setting.PostVideoCoinCount > 0 &&
		setting.VideoSize > 0 &&
		setting.VideoPCount > 0 &&
		setting.VideoCount > 0 &&
		setting.CommentCount > 0 &&
		setting.DanmuCount > 0 &&
		setting.DanmuUserRateCount > 0 &&
		setting.DanmuUserRateSeconds > 0 &&
		setting.DanmuUserVideoRateCount > 0 &&
		setting.DanmuUserVideoRateSeconds > 0 &&
		setting.DanmuIPRateCount > 0 &&
		setting.DanmuIPRateSeconds > 0 &&
		setting.DanmuVideoRateCount > 0 &&
		setting.DanmuVideoRateSeconds > 0
}
