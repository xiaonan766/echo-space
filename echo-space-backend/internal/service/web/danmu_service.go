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
	VideoExists(ctx context.Context, videoID string) (bool, error)
	ListDanmu(ctx context.Context, videoID string, fileID string) ([]domain.WebDanmuItem, error)
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

type LoadDanmuInput struct {
	FileID  string
	VideoID string
}

func NewDanmuService(repository DanmuRepository, settingStore DanmuSettingStore, limiter DanmuLimiter) *DanmuService {
	return &DanmuService{
		repository: repository, settingStore: settingStore, limiter: limiter, now: time.Now,
	}
}

func (s *DanmuService) LoadDanmu(ctx context.Context, input LoadDanmuInput) ([]domain.WebDanmuItem, error) {
	input = normalizeLoadDanmuInput(input)
	if err := validateLoadDanmuInput(input); err != nil {
		return nil, err
	}
	if s == nil || s.repository == nil {
		return nil, errors.New("danmu service is not ready")
	}

	exists, err := s.repository.VideoExists(ctx, input.VideoID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []domain.WebDanmuItem{}, nil
	}

	list, err := s.repository.ListDanmu(ctx, input.VideoID, input.FileID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.WebDanmuItem{}
	}
	return list, nil
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
		return &BusinessError{Info: "\u89c6\u9891\u6587\u4ef6\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return err
	}
	if target == nil {
		return &BusinessError{Info: "\u89c6\u9891\u6587\u4ef6\u4e0d\u5b58\u5728"}
	}
	if strings.Contains(target.Interaction, "0") {
		return &BusinessError{Info: "\u89c6\u9891\u5df2\u5173\u95ed\u5f39\u5e55"}
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
			return &BusinessError{Info: "\u53d1\u9001\u592a\u9891\u7e41\uff0c\u8bf7\u7a0d\u540e\u518d\u8bd5"}
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

func normalizeLoadDanmuInput(input LoadDanmuInput) LoadDanmuInput {
	input.FileID = strings.TrimSpace(input.FileID)
	input.VideoID = strings.TrimSpace(input.VideoID)
	return input
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

func validateLoadDanmuInput(input LoadDanmuInput) error {
	if len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID) {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if len(input.FileID) != videoFileIDLength || !isAlphaNumeric(input.FileID) {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	return nil
}

func validatePostDanmuInput(input PostDanmuInput) error {
	if input.UserID == "" {
		return &BusinessError{Info: "\u8bf7\u5148\u767b\u5f55"}
	}
	if input.Text == "" || utf8.RuneCountInString(input.Text) > maxDanmuTextLength {
		return &BusinessError{Info: "\u5f39\u5e55\u5185\u5bb9\u4e0d\u80fd\u4e3a\u7a7a\u4e14\u4e0d\u80fd\u8d85\u8fc7200\u4e2a\u5b57"}
	}
	if input.Mode < 0 || input.Time < 0 {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if input.Color == "" || utf8.RuneCountInString(input.Color) > maxDanmuColorLength {
		return &BusinessError{Info: "\u5f39\u5e55\u989c\u8272\u4e0d\u6b63\u786e"}
	}
	if len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID) {
		return &BusinessError{Info: "\u89c6\u9891ID\u4e0d\u6b63\u786e"}
	}
	if len(input.FileID) != videoFileIDLength || !isAlphaNumeric(input.FileID) {
		return &BusinessError{Info: "\u89c6\u9891\u6587\u4ef6ID\u4e0d\u6b63\u786e"}
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
