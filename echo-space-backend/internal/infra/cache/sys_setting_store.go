package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type SysSettingStore struct {
	client  *redis.Client
	key     string
	timeout time.Duration
}

func NewSysSettingStore(client *redis.Client, key string) *SysSettingStore {
	return &SysSettingStore{
		client:  client,
		key:     key,
		timeout: 3 * time.Second,
	}
}

func (s *SysSettingStore) Get(ctx context.Context) (domain.SysSetting, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	content, err := s.client.Get(ctx, s.key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return domain.SysSetting{}, false, nil
		}
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

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.client.Set(ctx, s.key, content, 0).Err()
}
