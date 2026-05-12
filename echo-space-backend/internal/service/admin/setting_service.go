package admin

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

const sysSettingKey = "echo-space:sys_setting"

type SettingService struct {
	settingStore *cache.SysSettingStore
}

func NewSettingService(redisClient *redis.Client) *SettingService {
	return &SettingService{
		settingStore: cache.NewSysSettingStore(redisClient, sysSettingKey),
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
	return setting, nil
}

func (s *SettingService) SaveSetting(ctx context.Context, setting domain.SysSetting) error {
	if !validSysSetting(setting) {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
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
		setting.DanmuCount > 0
}
