package web

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	checkCodeKeyPrefix     = "echo-space:check_code:web:"
	checkCodeTTL           = 5 * time.Minute
	defaultRegisterCoinCnt = 0
	userStatusEnable       = 1
	userSexSecrecy         = 2
	defaultUserTheme       = 1
)

var (
	emailRegexp = regexp.MustCompile(`^[\w-]+(\.[\w-]+)*@[\w-]+(\.[\w-]+)+$`)
)

type AccountService struct {
	captcha        *base64Captcha.Captcha
	userRepository *repository.UserRepository
}

type CheckCodeResult struct {
	CheckCode    string `json:"checkCode"`
	CheckCodeKey string `json:"checkCodeKey"`
}

type RegisterInput struct {
	Email            string
	NickName         string
	RegisterPassword string
	CheckCodeKey     string
	CheckCode        string
}

func NewAccountService(redisClient *redis.Client, userRepository *repository.UserRepository) *AccountService {
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
		captcha:        base64Captcha.NewCaptcha(driver, store),
		userRepository: userRepository,
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

func (s *AccountService) Register(ctx context.Context, input RegisterInput) error {
	input.Email = strings.TrimSpace(input.Email)
	input.NickName = strings.TrimSpace(input.NickName)
	input.RegisterPassword = strings.TrimSpace(input.RegisterPassword)

	if !s.captcha.Verify(input.CheckCodeKey, input.CheckCode, true) {
		return &BusinessError{Info: "\u56fe\u7247\u9a8c\u8bc1\u7801\u4e0d\u6b63\u786e"}
	}
	if err := validateRegisterInput(input); err != nil {
		return err
	}

	if _, err := s.userRepository.FindByEmail(ctx, input.Email); err == nil {
		return &BusinessError{Info: "\u90ae\u7bb1\u8d26\u53f7\u5df2\u5b58\u5728"}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if _, err := s.userRepository.FindByNickName(ctx, input.NickName); err == nil {
		return &BusinessError{Info: "\u6635\u79f0\u5df2\u5b58\u5728"}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	userID, err := s.generateUniqueUserID(ctx)
	if err != nil {
		return err
	}

	user := &domain.UserInfo{
		UserID:           userID,
		NickName:         input.NickName,
		Email:            input.Email,
		Password:         md5Hex(input.RegisterPassword),
		Sex:              userSexSecrecy,
		JoinTime:         time.Now(),
		Status:           userStatusEnable,
		TotalCoinCount:   defaultRegisterCoinCnt,
		CurrentCoinCount: defaultRegisterCoinCnt,
		Theme:            defaultUserTheme,
		Avatar:           "",
	}
	if err := s.userRepository.Create(ctx, user); err != nil {
		return err
	}

	return nil
}

func validateRegisterInput(input RegisterInput) error {
	if input.Email == "" || len(input.Email) > 150 || !emailRegexp.MatchString(input.Email) {
		return &BusinessError{Info: "\u8bf7\u8f93\u5165\u6b63\u786e\u7684\u90ae\u7bb1"}
	}
	if input.NickName == "" || len([]rune(input.NickName)) > 20 {
		return &BusinessError{Info: "\u8bf7\u8f93\u5165\u6635\u79f0"}
	}
	if !validPassword(input.RegisterPassword) {
		return &BusinessError{Info: "\u5bc6\u7801\u53ea\u80fd\u662f\u6570\u5b57\uff0c\u5b57\u6bcd\uff0c\u7279\u6b8a\u5b57\u7b26 8-18 \u4f4d"}
	}
	return nil
}

func validPassword(password string) bool {
	if len(password) < 8 || len(password) > 18 {
		return false
	}

	hasLetter := false
	hasDigit := false
	for _, char := range password {
		switch {
		case char >= '0' && char <= '9':
			hasDigit = true
		case char >= 'a' && char <= 'z':
			hasLetter = true
		case char >= 'A' && char <= 'Z':
			hasLetter = true
		case strings.ContainsRune("~!@#$%^&*_", char):
		default:
			return false
		}
	}
	return hasLetter && hasDigit
}

func (s *AccountService) generateUniqueUserID(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		userID, err := randomNumberString(10)
		if err != nil {
			return "", err
		}

		exists, err := s.userRepository.ExistsByUserID(ctx, userID)
		if err != nil {
			return "", err
		}
		if !exists {
			return userID, nil
		}
	}
	return "", fmt.Errorf("generate unique user id failed")
}

func randomNumberString(length int) (string, error) {
	builder := strings.Builder{}
	builder.Grow(length)

	for i := 0; i < length; i++ {
		number, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		builder.WriteByte(byte('0' + number.Int64()))
	}
	return builder.String(), nil
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}
