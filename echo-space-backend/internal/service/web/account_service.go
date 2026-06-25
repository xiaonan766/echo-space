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
	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	checkCodeKeyPrefix     = "echo-space:check_code:web:"
	checkCodeTTL           = 5 * time.Minute
	webTokenKeyPrefix      = "echo-space:token:web:"
	webTokenTTL            = 24 * time.Hour
	WebTokenCookieName     = "token"
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
	tokenStore     *cache.TokenStore
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

type LoginInput struct {
	Email        string
	Password     string
	CheckCodeKey string
	CheckCode    string
	OldToken     string
	LoginIP      string
}

type TokenUserInfo struct {
	Token            string `json:"token"`
	UserID           string `json:"userId"`
	NickName         string `json:"nickName"`
	Avatar           string `json:"avatar"`
	Sex              int    `json:"sex"`
	Theme            int    `json:"theme"`
	CurrentCoinCount int    `json:"currentCoinCount"`
	ExpireAt         int64  `json:"expireAt"`
}

type UserCountInfo struct {
	FansCount        int `json:"fansCount"`
	FocusCount       int `json:"focusCount"`
	CurrentCoinCount int `json:"currentCoinCount"`
}

func NewAccountService(hybridCache *cache.HybridCache, userRepository *repository.UserRepository) *AccountService {
	store := cache.NewCaptchaStore(hybridCache, checkCodeKeyPrefix, checkCodeTTL)
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
		tokenStore:     cache.NewTokenStore(hybridCache, webTokenKeyPrefix),
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

func (s *AccountService) Login(ctx context.Context, input LoginInput) (*TokenUserInfo, error) {
	input.Email = strings.TrimSpace(input.Email)
	input.Password = strings.TrimSpace(input.Password)
	input.LoginIP = normalizeLoginIP(input.LoginIP)

	if strings.TrimSpace(input.OldToken) != "" {
		defer func() {
			_ = s.tokenStore.Delete(context.Background(), input.OldToken)
		}()
	}

	if !s.captcha.Verify(input.CheckCodeKey, input.CheckCode, true) {
		return nil, &BusinessError{Info: "\u56fe\u7247\u9a8c\u8bc1\u7801\u4e0d\u6b63\u786e"}
	}
	if input.Email == "" || !emailRegexp.MatchString(input.Email) || input.Password == "" {
		return nil, &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	user, err := s.userRepository.FindByEmail(ctx, input.Email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "\u8d26\u53f7\u6216\u5bc6\u7801\u9519\u8bef"}
	}
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(user.Password, input.Password) {
		return nil, &BusinessError{Info: "\u8d26\u53f7\u6216\u5bc6\u7801\u9519\u8bef"}
	}
	if user.Status == 0 {
		return nil, &BusinessError{Info: "\u8d26\u53f7\u5df2\u7981\u7528"}
	}

	token, err := randomHexToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.userRepository.UpdateLoginInfo(ctx, user.UserID, now, input.LoginIP); err != nil {
		return nil, err
	}

	result := &TokenUserInfo{
		Token:            token,
		UserID:           user.UserID,
		NickName:         user.NickName,
		Avatar:           user.Avatar,
		Sex:              user.Sex,
		Theme:            user.Theme,
		CurrentCoinCount: user.CurrentCoinCount,
		ExpireAt:         now.Add(webTokenTTL).UnixMilli(),
	}
	if err := s.tokenStore.Set(ctx, token, result, webTokenTTL); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *AccountService) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	return s.tokenStore.Delete(ctx, token)
}

func (s *AccountService) AutoLogin(ctx context.Context, token string) (*TokenUserInfo, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}

	info, ok, err := s.GetTokenUserInfo(ctx, token)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	info.Token = token
	info.ExpireAt = time.Now().Add(webTokenTTL).UnixMilli()
	if err := s.tokenStore.Set(ctx, token, info, webTokenTTL); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *AccountService) GetUserCountInfo(ctx context.Context, userID string) (*UserCountInfo, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}

	user, err := s.userRepository.FindByUserID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &BusinessError{Info: "\u7528\u6237\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return nil, err
	}

	focusCount, err := s.userRepository.CountFocus(ctx, userID)
	if err != nil {
		return nil, err
	}
	fansCount, err := s.userRepository.CountFans(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserCountInfo{
		FansCount:        int(fansCount),
		FocusCount:       int(focusCount),
		CurrentCoinCount: user.CurrentCoinCount,
	}, nil
}

func (s *AccountService) GetTokenUserInfo(ctx context.Context, token string) (*TokenUserInfo, bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, false, nil
	}

	var info TokenUserInfo
	ok, err := s.tokenStore.Get(ctx, token, &info, webTokenTTL)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &info, true, nil
}

func (s *AccountService) UpdateTokenUserInfo(ctx context.Context, token string, userInfo *domain.UserInfo) error {
	token = strings.TrimSpace(token)
	if token == "" || userInfo == nil {
		return nil
	}

	info, ok, err := s.GetTokenUserInfo(ctx, token)
	if err != nil || !ok {
		return err
	}

	info.NickName = userInfo.NickName
	info.Avatar = userInfo.Avatar
	info.Sex = userInfo.Sex
	info.Theme = userInfo.Theme
	info.CurrentCoinCount = userInfo.CurrentCoinCount
	info.ExpireAt = time.Now().Add(webTokenTTL).UnixMilli()
	return s.tokenStore.Set(ctx, token, info, webTokenTTL)
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

func randomHexToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func normalizeLoginIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if len(ip) <= 15 {
		return ip
	}
	return ip[:15]
}
