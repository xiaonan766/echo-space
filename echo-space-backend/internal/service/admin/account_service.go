package admin

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

const (
	checkCodeKeyPrefix   = "echo-space:check_code:admin:"
	checkCodeTTL         = 5 * time.Minute
	adminTokenKeyPrefix  = "echo-space:token:admin:"
	AdminTokenCookieName = "adminToken"
)

type AccountService struct {
	adminConfig config.AdminConfig
	captcha     *base64Captcha.Captcha
	tokenStore  *cache.TokenStore
}

type CheckCodeResult struct {
	CheckCode    string `json:"checkCode"`
	CheckCodeKey string `json:"checkCodeKey"`
}

type LoginInput struct {
	Account      string
	Password     string
	CheckCodeKey string
	CheckCode    string
	OldToken     string
	LoginIP      string
}

type LoginResult struct {
	Account string
	Token   string
}

type tokenInfo struct {
	Account string `json:"account"`
	LoginIP string `json:"loginIp"`
	LoginAt string `json:"loginAt"`
}

type BusinessError struct {
	Info string
}

func (e *BusinessError) Error() string {
	return e.Info
}

func NewAccountService(redisClient *redis.Client, adminConfig config.AdminConfig) *AccountService {
	store := cache.NewCaptchaStore(redisClient, checkCodeKeyPrefix, checkCodeTTL)
	driver := base64Captcha.NewDriverMath(
		42,
		100,
		0,
		base64Captcha.OptionShowHollowLine|base64Captcha.OptionShowSlimeLine,
		&color.RGBA{R: 255, G: 255, B: 255, A: 255},
		nil,
		nil,
	)

	return &AccountService{
		adminConfig: adminConfig,
		captcha:     base64Captcha.NewCaptcha(driver, store),
		tokenStore:  cache.NewTokenStore(redisClient, adminTokenKeyPrefix),
	}
}

func (s *AccountService) GenerateCheckCode() (*CheckCodeResult, error) {
	checkCodeKey, checkCodeBase64, _, err := s.captcha.Generate()
	if err != nil {
		return nil, err
	}

	return &CheckCodeResult{
		CheckCode:    checkCodeBase64,
		CheckCodeKey: checkCodeKey,
	}, nil
}

func (s *AccountService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	if strings.TrimSpace(input.OldToken) != "" {
		defer func() {
			_ = s.tokenStore.Delete(context.Background(), input.OldToken)
		}()
	}

	if !s.captcha.Verify(input.CheckCodeKey, input.CheckCode, true) {
		return nil, &BusinessError{Info: "\u56fe\u7247\u9a8c\u8bc1\u7801\u4e0d\u6b63\u786e"}
	}

	if input.Account != s.adminConfig.Account || !strings.EqualFold(input.Password, md5Hex(s.adminConfig.Password)) {
		return nil, &BusinessError{Info: "\u8d26\u53f7\u6216\u5bc6\u7801\u9519\u8bef"}
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate admin token: %w", err)
	}

	info := tokenInfo{
		Account: input.Account,
		LoginIP: input.LoginIP,
		LoginAt: time.Now().Format(time.RFC3339),
	}
	if err := s.tokenStore.Set(ctx, token, info, s.adminConfig.TokenTTLDuration()); err != nil {
		return nil, fmt.Errorf("save admin token: %w", err)
	}

	return &LoginResult{
		Account: input.Account,
		Token:   token,
	}, nil
}

func IsBusinessError(err error) (*BusinessError, bool) {
	var businessError *BusinessError
	if errors.As(err, &businessError) {
		return businessError, true
	}
	return nil, false
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func generateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
