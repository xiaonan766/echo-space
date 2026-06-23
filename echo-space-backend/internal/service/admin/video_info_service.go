package admin

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

type VideoInfoService struct {
	videoPostRepository *repository.VideoPostRepository
	settingStore        *cache.SysSettingStore
}

type VideoInfoListInput struct {
	PageNo         int
	PageSize       int
	VideoNameFuzzy string
	PCategoryID    int
	CategoryID     int
	Status         *int
	RecommendType  *int
}

type AuditVideoInput struct {
	VideoID string
	Status  int
	Reason  string
}

func NewVideoInfoService(videoPostRepository *repository.VideoPostRepository, settingStore ...*cache.SysSettingStore) *VideoInfoService {
	var store *cache.SysSettingStore
	if len(settingStore) > 0 {
		store = settingStore[0]
	}
	return &VideoInfoService{
		videoPostRepository: videoPostRepository,
		settingStore:        store,
	}
}

func (s *VideoInfoService) LoadVideoList(ctx context.Context, input VideoInfoListInput) (domain.PaginationResult[domain.AdminVideoPostItem], error) {
	input = normalizeVideoInfoListInput(input)
	list, totalCount, err := s.videoPostRepository.ListAdminPostsByPage(ctx, repository.AdminVideoPostListQuery{
		PageNo:         input.PageNo,
		PageSize:       input.PageSize,
		VideoNameFuzzy: input.VideoNameFuzzy,
		PCategoryID:    input.PCategoryID,
		CategoryID:     input.CategoryID,
		Status:         input.Status,
		RecommendType:  input.RecommendType,
	})
	if err != nil {
		return domain.PaginationResult[domain.AdminVideoPostItem]{}, err
	}

	fillVideoPostStatusNames(list)
	return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
}

func (s *VideoInfoService) AuditVideo(ctx context.Context, input AuditVideoInput) error {
	input.VideoID = strings.TrimSpace(input.VideoID)
	input.Reason = strings.TrimSpace(input.Reason)
	if len(input.VideoID) != 10 || !isAlphaNumeric(input.VideoID) || !isValidAuditVideoStatus(input.Status) {
		return &BusinessError{Info: "\u53c2\u6570\u9519\u8bef"}
	}
	if input.Status == domain.VideoPostStatusRejected {
		if input.Reason == "" {
			return &BusinessError{Info: "\u8bf7\u8f93\u5165\u62d2\u7edd\u7406\u7531"}
		}
		if utf8.RuneCountInString(input.Reason) > 200 {
			return &BusinessError{Info: "\u62d2\u7edd\u7406\u7531\u4e0d\u80fd\u8d85\u8fc7200\u4e2a\u5b57\u7b26"}
		}
	}

	setting := domain.DefaultSysSetting()
	if s.settingStore != nil {
		stored, exists, err := s.settingStore.Get(ctx)
		if err != nil {
			return err
		}
		if exists {
			setting = stored
		}
	}

	err := s.videoPostRepository.AuditVideo(ctx, repository.AuditVideoData{
		VideoID:            input.VideoID,
		Status:             input.Status,
		PostVideoCoinCount: setting.PostVideoCoinCount,
	})
	if errors.Is(err, repository.ErrVideoAuditConflict) {
		return &BusinessError{Info: "\u5ba1\u6838\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5"}
	}
	if errors.Is(err, repository.ErrVideoNoPublishableFiles) {
		return &BusinessError{Info: "\u89c6\u9891\u6587\u4ef6\u4e0d\u5b58\u5728\u6216\u672a\u8f6c\u7801\u5b8c\u6210"}
	}
	return err
}

func normalizeVideoInfoListInput(input VideoInfoListInput) VideoInfoListInput {
	if input.PageNo <= 0 {
		input.PageNo = defaultPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultPageSize
	}
	if input.PageSize > maxPageSize {
		input.PageSize = maxPageSize
	}
	input.VideoNameFuzzy = strings.TrimSpace(input.VideoNameFuzzy)
	if input.PCategoryID < 0 {
		input.PCategoryID = 0
	}
	if input.CategoryID < 0 {
		input.CategoryID = 0
	}
	if input.Status != nil && !isValidVideoPostStatus(*input.Status) {
		input.Status = nil
	}
	if input.RecommendType != nil && (*input.RecommendType != 0 && *input.RecommendType != 1) {
		input.RecommendType = nil
	}
	return input
}

func fillVideoPostStatusNames(list []domain.AdminVideoPostItem) {
	for index := range list {
		list[index].StatusName = videoPostStatusName(list[index].Status)
	}
}

func videoPostStatusName(status int) string {
	switch status {
	case domain.VideoPostStatusTranscoding:
		return "转码中"
	case domain.VideoPostStatusTransferFailed:
		return "转码失败"
	case domain.VideoPostStatusPendingReview:
		return "待审核"
	case domain.VideoPostStatusApproved:
		return "审核成功"
	case domain.VideoPostStatusRejected:
		return "审核失败"
	default:
		return "未知状态"
	}
}

func isValidVideoPostStatus(status int) bool {
	return status >= domain.VideoPostStatusTranscoding && status <= domain.VideoPostStatusRejected
}

func IsValidVideoPostStatusForAdmin(status int) bool {
	return isValidVideoPostStatus(status)
}

func isValidAuditVideoStatus(status int) bool {
	return status == domain.VideoPostStatusApproved || status == domain.VideoPostStatusRejected
}

func isAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char >= '0' && char <= '9' {
			continue
		}
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		return false
	}
	return true
}
