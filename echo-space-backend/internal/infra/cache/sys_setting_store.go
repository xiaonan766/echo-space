package cache

import (
	"context"
	"encoding/json"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type SysSettingStore struct {
	cache *HybridCache
	key   string
}

func NewSysSettingStore(cache *HybridCache, key string) *SysSettingStore {
	return &SysSettingStore{
		cache: cache,
		key:   key,
	}
}

func (s *SysSettingStore) Get(ctx context.Context) (domain.SysSetting, bool, error) {
	content, ok, err := s.cache.Get(ctx, s.key, 0, true)
	if err != nil || !ok {
		return domain.SysSetting{}, false, err
	}

	var setting domain.SysSetting
	if err := json.Unmarshal(content, &setting); err != nil {
		return domain.SysSetting{}, false, err
	}
	return setting, true, nil
}

func (s *SysSettingStore) Save(ctx context.Context, setting domain.SysSetting) error {
	content, err := json.Marshal(setting)
	if err != nil {
		return err
	}

	return s.cache.Set(ctx, s.key, content, 0, RecoverWriteBack)
}
