package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

var ErrVideoPostNotEditable = errors.New("video post is not editable")
var ErrVideoTranscodeClaimLost = errors.New("video transcode claim was lost")

type VideoPostRepository struct {
	db *gorm.DB
}

type SaveNewVideoPostData struct {
	Post     domain.VideoInfoPost
	Files    []domain.VideoInfoFilePost
	Messages []domain.VideoTranscodeMessageRecord
}

type SaveEditedVideoPostData struct {
	Post           domain.VideoInfoPost
	Files          []domain.VideoInfoFilePost
	DeletedFileIDs []string
	Messages       []domain.VideoTranscodeMessageRecord
}

type VideoTranscodeResult struct {
	FileSize int64
	FilePath string
	Duration int
}

func NewVideoPostRepository(db *gorm.DB) *VideoPostRepository {
	return &VideoPostRepository{db: db}
}

func (r *VideoPostRepository) FindPostWithFiles(ctx context.Context, videoID string, userID string) (*domain.VideoInfoPost, []domain.VideoInfoFilePost, error) {
	var post domain.VideoInfoPost
	if err := r.db.WithContext(ctx).
		Where("video_id = ? AND user_id = ?", videoID, userID).
		Take(&post).Error; err != nil {
		return nil, nil, err
	}

	var files []domain.VideoInfoFilePost
	if err := r.db.WithContext(ctx).
		Where("video_id = ? AND user_id = ?", videoID, userID).
		Order("file_index asc").
		Find(&files).Error; err != nil {
		return nil, nil, err
	}
	return &post, files, nil
}

func (r *VideoPostRepository) CreatePost(ctx context.Context, data SaveNewVideoPostData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&data.Post).Error; err != nil {
			return err
		}
		if len(data.Files) > 0 {
			if err := tx.Create(&data.Files).Error; err != nil {
				return err
			}
		}
		if len(data.Messages) > 0 {
			return tx.Create(&data.Messages).Error
		}
		return nil
	})
}

func (r *VideoPostRepository) UpdatePost(ctx context.Context, data SaveEditedVideoPostData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current domain.VideoInfoPost
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("video_id = ? AND user_id = ?", data.Post.VideoID, data.Post.UserID).
			Take(&current).Error; err != nil {
			return err
		}
		if current.Status == domain.VideoPostStatusTranscoding || current.Status == domain.VideoPostStatusPendingReview {
			return ErrVideoPostNotEditable
		}

		updates := map[string]any{
			"video_cover":      data.Post.VideoCover,
			"video_name":       data.Post.VideoName,
			"last_update_time": data.Post.LastUpdateTime,
			"p_category_id":    data.Post.PCategoryID,
			"category_id":      data.Post.CategoryID,
			"status":           data.Post.Status,
			"post_type":        data.Post.PostType,
			"origin_info":      data.Post.OriginInfo,
			"tags":             data.Post.Tags,
			"introduction":     data.Post.Introduction,
			"interaction":      data.Post.Interaction,
		}
		if data.Post.Status == domain.VideoPostStatusTranscoding {
			updates["duration"] = nil
		}
		if err := tx.Model(&domain.VideoInfoPost{}).
			Where("video_id = ? AND user_id = ?", data.Post.VideoID, data.Post.UserID).
			Updates(updates).Error; err != nil {
			return err
		}

		for _, file := range data.Files {
			if strings.TrimSpace(file.FileID) == "" {
				return errors.New("video file id is empty")
			}
			var count int64
			if err := tx.Model(&domain.VideoInfoFilePost{}).
				Where("file_id = ? AND video_id = ? AND user_id = ?", file.FileID, data.Post.VideoID, data.Post.UserID).
				Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				if err := tx.Create(&file).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Model(&domain.VideoInfoFilePost{}).
				Where("file_id = ? AND video_id = ? AND user_id = ?", file.FileID, data.Post.VideoID, data.Post.UserID).
				Updates(map[string]any{
					"file_index":  file.FileIndex,
					"file_name":   file.FileName,
					"update_type": file.UpdateType,
				}).Error; err != nil {
				return err
			}
		}

		if len(data.DeletedFileIDs) > 0 {
			if err := tx.Model(&domain.VideoInfoFilePost{}).
				Where("video_id = ? AND user_id = ? AND file_id IN ?", data.Post.VideoID, data.Post.UserID, data.DeletedFileIDs).
				Update("update_type", domain.VideoFileUpdateDeletePending).Error; err != nil {
				return err
			}
		}
		if len(data.Messages) > 0 {
			return tx.Create(&data.Messages).Error
		}
		return nil
	})
}

func (r *VideoPostRepository) ListTranscodeMessagesForPublish(ctx context.Context, limit int, publishedBefore time.Time) ([]domain.VideoTranscodeMessageRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	now := time.Now()
	var list []domain.VideoTranscodeMessageRecord
	err := r.db.WithContext(ctx).
		Where(`
			(message_status IN ? AND (next_retry_time IS NULL OR next_retry_time <= ?))
			OR (message_status = ? AND update_time <= ?)
			OR (message_status = ? AND locked_until <= ?)
		`, []int{domain.VideoTranscodeMessageWaitPublish, domain.VideoTranscodeMessageRetryWait}, now,
			domain.VideoTranscodeMessagePublished, publishedBefore,
			domain.VideoTranscodeMessageProcessing, now).
		Order("create_time asc").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (r *VideoPostRepository) MarkTranscodeMessagePublished(ctx context.Context, messageID string, publishedBefore time.Time) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.VideoTranscodeMessageRecord{}).
		Where("message_id = ?", messageID).
		Where(`
			message_status IN ?
			OR (message_status = ? AND update_time <= ?)
			OR (message_status = ? AND locked_until <= ?)
		`, []int{domain.VideoTranscodeMessageWaitPublish, domain.VideoTranscodeMessageRetryWait, domain.VideoTranscodeMessagePublished},
			domain.VideoTranscodeMessagePublished, publishedBefore,
			domain.VideoTranscodeMessageProcessing, now).
		Updates(map[string]any{
			"message_status":  domain.VideoTranscodeMessagePublished,
			"next_retry_time": nil,
			"locked_until":    nil,
			"lock_token":      "",
			"last_error":      "",
		}).Error
}

func (r *VideoPostRepository) DelayTranscodeMessagePublish(ctx context.Context, messageID string, nextRetry time.Time, cause error) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.VideoTranscodeMessageRecord{}).
		Where("message_id = ?", messageID).
		Where("message_status IN ? OR (message_status = ? AND locked_until <= ?)",
			[]int{domain.VideoTranscodeMessageWaitPublish, domain.VideoTranscodeMessageRetryWait, domain.VideoTranscodeMessagePublished},
			domain.VideoTranscodeMessageProcessing, now).
		Updates(map[string]any{
			"message_status":  domain.VideoTranscodeMessageWaitPublish,
			"next_retry_time": nextRetry,
			"locked_until":    nil,
			"lock_token":      "",
			"last_error":      trimVideoMessageError(cause),
		}).Error
}

func (r *VideoPostRepository) ClaimTranscodeMessage(ctx context.Context, messageID string, lockToken string, leaseUntil time.Time) (*domain.VideoTranscodeMessageRecord, bool, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&domain.VideoTranscodeMessageRecord{}).
		Where("message_id = ?", messageID).
		Where(`
			(message_status IN ? AND (next_retry_time IS NULL OR next_retry_time <= ?))
			OR (message_status = ? AND locked_until <= ?)
		`, []int{domain.VideoTranscodeMessageWaitPublish, domain.VideoTranscodeMessagePublished, domain.VideoTranscodeMessageRetryWait}, now,
			domain.VideoTranscodeMessageProcessing, now).
		Updates(map[string]any{
			"message_status": domain.VideoTranscodeMessageProcessing,
			"locked_until":   leaseUntil,
			"lock_token":     lockToken,
		})
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}

	var message domain.VideoTranscodeMessageRecord
	if err := r.db.WithContext(ctx).Where("message_id = ?", messageID).Take(&message).Error; err != nil {
		return nil, false, err
	}
	return &message, true, nil
}

func (r *VideoPostRepository) RefreshTranscodeLease(ctx context.Context, messageID string, lockToken string, leaseUntil time.Time) error {
	result := r.db.WithContext(ctx).Model(&domain.VideoTranscodeMessageRecord{}).
		Where("message_id = ? AND message_status = ? AND lock_token = ?", messageID, domain.VideoTranscodeMessageProcessing, lockToken).
		Update("locked_until", leaseUntil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVideoTranscodeClaimLost
	}
	return nil
}

func (r *VideoPostRepository) CompleteTranscodeMessage(ctx context.Context, messageID string, fileID string, lockToken string, result VideoTranscodeResult) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		messageResult := tx.Model(&domain.VideoTranscodeMessageRecord{}).
			Where("message_id = ? AND message_status = ? AND lock_token = ?", messageID, domain.VideoTranscodeMessageProcessing, lockToken).
			Updates(map[string]any{
				"message_status":  domain.VideoTranscodeMessageSuccess,
				"next_retry_time": nil,
				"locked_until":    nil,
				"lock_token":      "",
				"last_error":      "",
			})
		if messageResult.Error != nil {
			return messageResult.Error
		}
		if messageResult.RowsAffected == 0 {
			return ErrVideoTranscodeClaimLost
		}
		if err := tx.Model(&domain.VideoInfoFilePost{}).
			Where("file_id = ?", fileID).
			Updates(map[string]any{
				"file_size":       result.FileSize,
				"file_path":       result.FilePath,
				"duration":        result.Duration,
				"transfer_result": domain.VideoFileTransferSuccess,
			}).Error; err != nil {
			return err
		}
		return refreshVideoPostTransferStatus(tx, fileID)
	})
}

func (r *VideoPostRepository) RetryOrFailTranscodeMessage(ctx context.Context, messageID string, fileID string, lockToken string, maxRetries int, nextRetry time.Time, cause error) (bool, error) {
	dead := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var message domain.VideoTranscodeMessageRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("message_id = ?", messageID).Take(&message).Error; err != nil {
			return err
		}
		if message.MessageStatus == domain.VideoTranscodeMessageSuccess || message.MessageStatus == domain.VideoTranscodeMessageDead {
			dead = message.MessageStatus == domain.VideoTranscodeMessageDead
			return nil
		}
		if lockToken != "" && (message.MessageStatus != domain.VideoTranscodeMessageProcessing || message.LockToken != lockToken) {
			return ErrVideoTranscodeClaimLost
		}
		if lockToken == "" {
			publishable := message.MessageStatus == domain.VideoTranscodeMessageWaitPublish ||
				message.MessageStatus == domain.VideoTranscodeMessageRetryWait ||
				message.MessageStatus == domain.VideoTranscodeMessagePublished ||
				(message.MessageStatus == domain.VideoTranscodeMessageProcessing && message.LockedUntil != nil && !message.LockedUntil.After(time.Now()))
			if !publishable {
				return ErrVideoTranscodeClaimLost
			}
		}

		retryCount := message.RetryCount + 1
		dead = retryCount >= maxRetries
		status := domain.VideoTranscodeMessageRetryWait
		var retryAt any = nextRetry
		if dead {
			status = domain.VideoTranscodeMessageDead
			retryAt = nil
		}
		if err := tx.Model(&domain.VideoTranscodeMessageRecord{}).
			Where("message_id = ?", messageID).
			Updates(map[string]any{
				"message_status":  status,
				"retry_count":     retryCount,
				"next_retry_time": retryAt,
				"locked_until":    nil,
				"lock_token":      "",
				"last_error":      trimVideoMessageError(cause),
			}).Error; err != nil {
			return err
		}
		if !dead {
			return nil
		}
		if err := tx.Model(&domain.VideoInfoFilePost{}).
			Where("file_id = ?", fileID).
			Update("transfer_result", domain.VideoFileTransferFailed).Error; err != nil {
			return err
		}
		return refreshVideoPostTransferStatus(tx, fileID)
	})
	return dead, err
}

func refreshVideoPostTransferStatus(tx *gorm.DB, fileID string) error {
	var file domain.VideoInfoFilePost
	if err := tx.Where("file_id = ?", fileID).Take(&file).Error; err != nil {
		return err
	}

	var failedCount int64
	if err := tx.Model(&domain.VideoInfoFilePost{}).
		Where("video_id = ? AND update_type <> ? AND transfer_result = ?", file.VideoID, domain.VideoFileUpdateDeletePending, domain.VideoFileTransferFailed).
		Count(&failedCount).Error; err != nil {
		return err
	}
	if failedCount > 0 {
		return tx.Model(&domain.VideoInfoPost{}).
			Where("video_id = ?", file.VideoID).
			Update("status", domain.VideoPostStatusTransferFailed).Error
	}

	var processingCount int64
	if err := tx.Model(&domain.VideoInfoFilePost{}).
		Where("video_id = ? AND update_type <> ? AND transfer_result = ?", file.VideoID, domain.VideoFileUpdateDeletePending, domain.VideoFileTransferProcessing).
		Count(&processingCount).Error; err != nil {
		return err
	}
	if processingCount > 0 {
		return nil
	}

	var duration int
	if err := tx.Model(&domain.VideoInfoFilePost{}).
		Where("video_id = ? AND update_type <> ?", file.VideoID, domain.VideoFileUpdateDeletePending).
		Select("COALESCE(SUM(duration), 0)").
		Scan(&duration).Error; err != nil {
		return err
	}
	return tx.Model(&domain.VideoInfoPost{}).
		Where("video_id = ?", file.VideoID).
		Updates(map[string]any{
			"status":   domain.VideoPostStatusPendingReview,
			"duration": duration,
		}).Error
}

func trimVideoMessageError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 500 {
		return message[:500]
	}
	return message
}
