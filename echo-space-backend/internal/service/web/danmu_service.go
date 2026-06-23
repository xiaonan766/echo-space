package web

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

const (
	maxDanmuTextLength  = 200
	maxDanmuColorLength = 10
)

type DanmuRepository interface {
	FindDanmuTarget(ctx context.Context, videoID string, fileID string) (*domain.DanmuTargetInfo, error)
	CreateDanmu(ctx context.Context, danmu domain.VideoDanmu) error
}

type DanmuSettingStore interface {
	Get(ctx context.Context) (domain.SysSetting, bool, error)
}

type DanmuLimiter interface {
	Allow(ctx context.Context, config cache.DanmuRateLimitConfig, userID string, videoID string, clientIP string) (bool, error)
}

type DanmuService struct {
	repository   DanmuRepository
	settingStore DanmuSettingStore
	limiter      DanmuLimiter
	now          func() time.Time
}

type PostDanmuInput struct {
	UserID   string
	Text     string
	Mode     int
	Color    string
	Time     int
	FileID   string
	VideoID  string
	ClientIP string
}

func NewDanmuService(repository DanmuRepository, settingStore DanmuSettingStore, limiter DanmuLimiter) *DanmuService {
	return &DanmuService{
		repository: repository, settingStore: settingStore, limiter: limiter, now: time.Now,
	}
}

func (s *DanmuService) PostDanmu(ctx context.Context, input PostDanmuInput) error {
	input = normalizePostDanmuInput(input)
	if err := validatePostDanmuInput(input); err != nil {
		return err
	}
	if s == nil || s.repository == nil {
		return errors.New("danmu service is not ready")
	}

	target, err := s.repository.FindDanmuTarget(ctx, input.VideoID, input.FileID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &BusinessError{Info: "视频文件不存在"}
	}
	if err != nil {
		return err
	}
	if target == nil {
		return &BusinessError{Info: "视频文件不存在"}
	}
	if strings.Contains(target.Interaction, "0") {
		return &BusinessError{Info: "视频已关闭弹幕"}
	}

	setting, err := s.getSetting(ctx)
	if err != nil {
		return fmt.Errorf("get danmu rate setting: %w", err)
	}
	if s.limiter != nil {
		allowed, err := s.limiter.Allow(ctx, buildDanmuRateLimitConfig(setting), input.UserID, input.VideoID, input.ClientIP)
		if err != nil {
			return fmt.Errorf("danmu rate limit: %w", err)
		}
		if !allowed {
			return &BusinessError{Info: "发送太频繁，请稍后再试"}
		}
	}

	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	return s.repository.CreateDanmu(ctx, domain.VideoDanmu{
		VideoID: input.VideoID, FileID: input.FileID, UserID: input.UserID, PostTime: now,
		Text: input.Text, Mode: input.Mode, Color: input.Color, Time: input.Time,
	})
}

func (s *DanmuService) getSetting(ctx context.Context) (domain.SysSetting, error) {
	setting := domain.DefaultSysSetting()
	if s == nil || s.settingStore == nil {
		return setting, nil
	}
	stored, exists, err := s.settingStore.Get(ctx)
	if err != nil {
		return domain.SysSetting{}, err
	}
	if exists {
		setting = stored
	}
	return domain.NormalizeSysSetting(setting), nil
}

func normalizePostDanmuInput(input PostDanmuInput) PostDanmuInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.Text = strings.TrimSpace(input.Text)
	input.Color = strings.TrimSpace(input.Color)
	input.FileID = strings.TrimSpace(input.FileID)
	input.VideoID = strings.TrimSpace(input.VideoID)
	input.ClientIP = strings.TrimSpace(input.ClientIP)
	if input.ClientIP == "" {
		input.ClientIP = "unknown"
	}
	return input
}

func validatePostDanmuInput(input PostDanmuInput) error {
	if input.UserID == "" {
		return &BusinessError{Info: "请先登录"}
	}
	if input.Text == "" || utf8.RuneCountInString(input.Text) > maxDanmuTextLength {
		return &BusinessError{Info: "弹幕内容不能为空且不能超过200个字"}
	}
	if input.Mode < 0 || input.Time < 0 {
		return &BusinessError{Info: "参数错误"}
	}
	if input.Color == "" || utf8.RuneCountInString(input.Color) > maxDanmuColorLength {
		return &BusinessError{Info: "弹幕颜色不正确"}
	}
	if len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID) {
		return &BusinessError{Info: "视频ID不正确"}
	}
	if len(input.FileID) != videoFileIDLength || !isAlphaNumeric(input.FileID) {
		return &BusinessError{Info: "视频文件ID不正确"}
	}
	return nil
}

func buildDanmuRateLimitConfig(setting domain.SysSetting) cache.DanmuRateLimitConfig {
	setting = domain.NormalizeSysSetting(setting)
	return cache.DanmuRateLimitConfig{
		User:      cache.TokenBucketRule{Capacity: setting.DanmuUserRateCount, Window: time.Duration(setting.DanmuUserRateSeconds) * time.Second},
		UserVideo: cache.TokenBucketRule{Capacity: setting.DanmuUserVideoRateCount, Window: time.Duration(setting.DanmuUserVideoRateSeconds) * time.Second},
		IP:        cache.TokenBucketRule{Capacity: setting.DanmuIPRateCount, Window: time.Duration(setting.DanmuIPRateSeconds) * time.Second},
		Video:     cache.TokenBucketRule{Capacity: setting.DanmuVideoRateCount, Window: time.Duration(setting.DanmuVideoRateSeconds) * time.Second},
	}
}
