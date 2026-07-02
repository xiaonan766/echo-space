package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

var ErrVideoPostNotEditable = errors.New("video post is not editable")
var ErrVideoTranscodeClaimLost = errors.New("video transcode claim was lost")
var ErrVideoAuditConflict = errors.New("video audit conflict")
var ErrVideoNoPublishableFiles = errors.New("video has no publishable files")
var ErrVideoInfoNotFound = errors.New("video info not found")

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

type AuditVideoData struct {
	VideoID            string
	Status             int
	PostVideoCoinCount int
}

type VideoTranscodeResult struct {
	FileSize int64
	FilePath string
	Duration int
}

type VideoDownloadGenerationJob struct {
	FileID         string `gorm:"column:file_id"`
	FileName       string `gorm:"column:file_name"`
	FilePath       string `gorm:"column:file_path"`
	VideoID        string `gorm:"column:video_id"`
	VideoName      string `gorm:"column:video_name"`
	UserID         string `gorm:"column:user_id"`
	NickName       string `gorm:"column:nick_name"`
	DownloadStatus int    `gorm:"column:download_status"`
}

type UcenterVideoPostListQuery struct {
	UserID         string
	PageNo         int
	PageSize       int
	VideoNameFuzzy string
	Status         *int
	ContentType    *int
}

type AdminVideoPostListQuery struct {
	PageNo         int
	PageSize       int
	VideoNameFuzzy string
	PCategoryID    int
	CategoryID     int
	Status         *int
	RecommendType  *int
	ContentType    *int
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

func (r *VideoPostRepository) FindPostByID(ctx context.Context, videoID string) (*domain.VideoInfoPost, error) {
	var post domain.VideoInfoPost
	if err := r.db.WithContext(ctx).
		Where("video_id = ?", videoID).
		Take(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *VideoPostRepository) ListPostFiles(ctx context.Context, videoID string) ([]domain.VideoInfoFilePostItem, error) {
	var files []domain.VideoInfoFilePostItem
	err := r.db.WithContext(ctx).
		Table("video_info_file_post").
		Select(`
			file_id,
			upload_id,
			user_id,
			video_id,
			file_index,
			COALESCE(file_name, '') AS file_name,
			COALESCE(file_size, 0) AS file_size,
			COALESCE(file_path, '') AS file_path,
			COALESCE(update_type, 0) AS update_type,
			COALESCE(transfer_result, 0) AS transfer_result,
			COALESCE(duration, 0) AS duration
		`).
		Where("video_id = ?", videoID).
		Order("file_index asc").
		Scan(&files).Error
	return files, err
}

func (r *VideoPostRepository) FindPostFileByFileID(ctx context.Context, fileID string) (*domain.VideoInfoFilePostItem, error) {
	var file domain.VideoInfoFilePostItem
	err := r.db.WithContext(ctx).
		Table("video_info_file_post").
		Select(`
			file_id,
			upload_id,
			user_id,
			video_id,
			file_index,
			COALESCE(file_name, '') AS file_name,
			COALESCE(file_size, 0) AS file_size,
			COALESCE(file_path, '') AS file_path,
			COALESCE(update_type, 0) AS update_type,
			COALESCE(transfer_result, 0) AS transfer_result,
			COALESCE(duration, 0) AS duration
		`).
		Where("file_id = ?", fileID).
		Take(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *VideoPostRepository) ListUserPostsByPage(ctx context.Context, query UcenterVideoPostListQuery) ([]domain.UcenterVideoPostItem, int64, error) {
	countQuery := applyUcenterVideoPostFilter(r.db.WithContext(ctx).Table("video_info_post v"), query)
	var totalCount int64
	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	listQuery := applyUcenterVideoPostFilter(r.db.WithContext(ctx).Table("video_info_post v"), query)
	var list []domain.UcenterVideoPostItem
	offset := (query.PageNo - 1) * query.PageSize
	err := listQuery.
		Select(`
			v.video_id,
			v.video_cover,
			v.video_name,
			v.user_id,
			COALESCE(u.nick_name, '') AS nick_name,
			COALESCE(u.avatar, '') AS avatar,
			DATE_FORMAT(v.create_time, '%Y-%m-%d %H:%i:%s') AS create_time,
			DATE_FORMAT(v.last_update_time, '%Y-%m-%d %H:%i:%s') AS last_update_time,
			v.p_category_id,
			v.category_id,
			COALESCE(v.content_type, 0) AS content_type,
			v.status,
			v.post_type,
			COALESCE(v.origin_info, '') AS origin_info,
			COALESCE(v.tags, '') AS tags,
			COALESCE(v.introduction, '') AS introduction,
			COALESCE(v.interaction, '') AS interaction,
			COALESCE(v.download_permission, 1) AS download_permission,
			COALESCE(v.duration, 0) AS duration,
			COALESCE(vi.play_count, 0) AS play_count,
			COALESCE(vi.like_count, 0) AS like_count,
			COALESCE(vi.danmu_count, 0) AS danmu_count,
			COALESCE(vi.comment_count, 0) AS comment_count,
			COALESCE(vi.coin_count, 0) AS coin_count,
			COALESCE(vi.collect_count, 0) AS collect_count
		`).
		Joins("LEFT JOIN user_info u ON u.user_id = v.user_id").
		Joins("LEFT JOIN video_info vi ON vi.video_id = v.video_id").
		Order("v.last_update_time desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, totalCount, nil
}

func applyUcenterVideoPostFilter(db *gorm.DB, query UcenterVideoPostListQuery) *gorm.DB {
	db = db.Where("v.user_id = ?", query.UserID)
	if strings.TrimSpace(query.VideoNameFuzzy) != "" {
		db = db.Where("v.video_name LIKE ?", "%"+strings.TrimSpace(query.VideoNameFuzzy)+"%")
	}
	if query.ContentType != nil {
		db = db.Where("COALESCE(v.content_type, 0) = ?", *query.ContentType)
	}
	if query.Status == nil {
		return db
	}
	if *query.Status == -1 {
		return db.Where("v.status IN ?", []int{
			domain.VideoPostStatusTranscoding,
			domain.VideoPostStatusTransferFailed,
			domain.VideoPostStatusPendingReview,
		})
	}
	return db.Where("v.status = ?", *query.Status)
}

func (r *VideoPostRepository) ListAdminPostsByPage(ctx context.Context, query AdminVideoPostListQuery) ([]domain.AdminVideoPostItem, int64, error) {
	countQuery := applyAdminVideoPostFilter(
		r.db.WithContext(ctx).
			Table("video_info_post v").
			Joins("LEFT JOIN video_info vi ON vi.video_id = v.video_id"),
		query,
	)
	var totalCount int64
	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	listQuery := applyAdminVideoPostFilter(
		r.db.WithContext(ctx).
			Table("video_info_post v").
			Joins("LEFT JOIN video_info vi ON vi.video_id = v.video_id"),
		query,
	)
	var list []domain.AdminVideoPostItem
	offset := (query.PageNo - 1) * query.PageSize
	err := listQuery.
		Select(`
			v.video_id,
			COALESCE(v.video_cover, '') AS video_cover,
			COALESCE(v.video_name, '') AS video_name,
			v.user_id,
			COALESCE(u.nick_name, '') AS nick_name,
			COALESCE(u.avatar, '') AS avatar,
			DATE_FORMAT(v.create_time, '%Y-%m-%d %H:%i:%s') AS create_time,
			DATE_FORMAT(v.last_update_time, '%Y-%m-%d %H:%i:%s') AS last_update_time,
			v.p_category_id,
			v.category_id,
			COALESCE(v.content_type, 0) AS content_type,
			v.status,
			v.post_type,
			COALESCE(v.origin_info, '') AS origin_info,
			COALESCE(v.tags, '') AS tags,
			COALESCE(v.introduction, '') AS introduction,
			COALESCE(v.interaction, '') AS interaction,
			COALESCE(v.download_permission, 1) AS download_permission,
			COALESCE(v.duration, 0) AS duration,
			COALESCE(vi.play_count, 0) AS play_count,
			COALESCE(vi.like_count, 0) AS like_count,
			COALESCE(vi.danmu_count, 0) AS danmu_count,
			COALESCE(vi.comment_count, 0) AS comment_count,
			COALESCE(vi.coin_count, 0) AS coin_count,
			COALESCE(vi.collect_count, 0) AS collect_count,
			COALESCE(vi.recommend_type, 0) AS recommend_type
		`).
		Joins("LEFT JOIN user_info u ON u.user_id = v.user_id").
		Order("v.last_update_time desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, totalCount, nil
}

func applyAdminVideoPostFilter(db *gorm.DB, query AdminVideoPostListQuery) *gorm.DB {
	if strings.TrimSpace(query.VideoNameFuzzy) != "" {
		db = db.Where("v.video_name LIKE ?", "%"+strings.TrimSpace(query.VideoNameFuzzy)+"%")
	}
	if query.PCategoryID > 0 {
		db = db.Where("v.p_category_id = ?", query.PCategoryID)
	}
	if query.CategoryID > 0 {
		db = db.Where("v.category_id = ?", query.CategoryID)
	}
	if query.Status != nil {
		db = db.Where("v.status = ?", *query.Status)
	}
	if query.RecommendType != nil {
		db = db.Where("COALESCE(vi.recommend_type, 0) = ?", *query.RecommendType)
	}
	if query.ContentType != nil {
		db = db.Where("COALESCE(v.content_type, 0) = ?", *query.ContentType)
	}
	return db
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

func (r *VideoPostRepository) AuditVideo(ctx context.Context, data AuditVideoData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&domain.VideoInfoPost{}).
			Where("video_id = ? AND status = ?", data.VideoID, domain.VideoPostStatusPendingReview).
			Update("status", data.Status)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrVideoAuditConflict
		}

		if err := tx.Model(&domain.VideoInfoFilePost{}).
			Where("video_id = ? AND update_type <> ?", data.VideoID, domain.VideoFileUpdateDeletePending).
			Update("update_type", domain.VideoFileUpdateNone).Error; err != nil {
			return err
		}

		if data.Status == domain.VideoPostStatusRejected {
			return nil
		}

		var post domain.VideoInfoPost
		if err := tx.Where("video_id = ?", data.VideoID).Take(&post).Error; err != nil {
			return err
		}
		if post.ContentType == domain.ContentTypeImage {
			return createDynamicEventMessage(tx, post, domain.DynamicEventTypeImage)
		}

		var postFiles []domain.VideoInfoFilePost
		if err := tx.
			Where("video_id = ? AND update_type <> ? AND transfer_result = ? AND COALESCE(file_path, '') <> ''",
				data.VideoID, domain.VideoFileUpdateDeletePending, domain.VideoFileTransferSuccess).
			Order("file_index asc").
			Find(&postFiles).Error; err != nil {
			return err
		}
		if len(postFiles) == 0 {
			return ErrVideoNoPublishableFiles
		}

		var existingVideo domain.VideoInfo
		err := tx.Where("video_id = ?", data.VideoID).Take(&existingVideo).Error
		firstPublish := false
		if errors.Is(err, gorm.ErrRecordNotFound) {
			firstPublish = true
		} else if err != nil {
			return err
		}

		if firstPublish && data.PostVideoCoinCount > 0 {
			coinResult := tx.Model(&domain.UserInfo{}).
				Where("user_id = ?", post.UserID).
				Updates(map[string]any{
					"total_coin_count":   gorm.Expr("total_coin_count + ?", data.PostVideoCoinCount),
					"current_coin_count": gorm.Expr("current_coin_count + ?", data.PostVideoCoinCount),
				})
			if coinResult.Error != nil {
				return coinResult.Error
			}
			if coinResult.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}

		video := buildVideoInfoFromPost(post)
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "video_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"video_cover", "video_name", "user_id", "create_time", "last_update_time",
				"p_category_id", "category_id", "post_type", "origin_info", "tags",
				"introduction", "interaction", "download_permission", "duration",
			}),
		}).Create(&video).Error; err != nil {
			return err
		}

		if err := tx.Where("video_id = ?", data.VideoID).Delete(&domain.VideoInfoFile{}).Error; err != nil {
			return err
		}

		files := make([]domain.VideoInfoFile, 0, len(postFiles))
		for _, postFile := range postFiles {
			files = append(files, buildVideoInfoFileFromPost(postFile))
		}
		if err := tx.Create(&files).Error; err != nil {
			return err
		}
		return createDynamicEventMessage(tx, post, domain.DynamicEventTypeVideo)
	})
}

func (r *VideoPostRepository) ToggleRecommendVideo(ctx context.Context, videoID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var video domain.VideoInfo
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("video_id", "recommend_type").
			Where("video_id = ?", videoID).
			Take(&video).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoInfoNotFound
		}
		if err != nil {
			return err
		}

		recommendType := 1
		if video.RecommendType == 1 {
			recommendType = 0
		}
		return tx.Model(&domain.VideoInfo{}).
			Where("video_id = ?", videoID).
			Update("recommend_type", recommendType).Error
	})
}

func (r *VideoPostRepository) FindVideoSearchDocumentByID(ctx context.Context, videoID string) (*domain.VideoSearchDocument, error) {
	var document domain.VideoSearchDocument
	err := r.db.WithContext(ctx).
		Table("video_info").
		Select(videoSearchDocumentSelectColumns).
		Where("video_id = ?", videoID).
		Take(&document).Error
	if err != nil {
		return nil, err
	}
	return &document, nil
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
			"video_cover":         data.Post.VideoCover,
			"video_name":          data.Post.VideoName,
			"last_update_time":    data.Post.LastUpdateTime,
			"p_category_id":       data.Post.PCategoryID,
			"category_id":         data.Post.CategoryID,
			"content_type":        data.Post.ContentType,
			"status":              data.Post.Status,
			"post_type":           data.Post.PostType,
			"origin_info":         data.Post.OriginInfo,
			"tags":                data.Post.Tags,
			"introduction":        data.Post.Introduction,
			"interaction":         data.Post.Interaction,
			"download_permission": data.Post.DownloadPermission,
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

func buildVideoInfoFromPost(post domain.VideoInfoPost) domain.VideoInfo {
	return domain.VideoInfo{
		VideoID: post.VideoID, VideoCover: post.VideoCover, VideoName: post.VideoName, UserID: post.UserID,
		CreateTime: post.CreateTime, LastUpdateTime: post.LastUpdateTime, PCategoryID: post.PCategoryID,
		CategoryID: post.CategoryID, PostType: post.PostType, OriginInfo: post.OriginInfo, Tags: post.Tags,
		Introduction: post.Introduction, Interaction: post.Interaction, DownloadPermission: post.DownloadPermission,
		Duration: post.Duration,
	}
}

func buildVideoInfoFileFromPost(file domain.VideoInfoFilePost) domain.VideoInfoFile {
	filePath := ""
	if file.FilePath != nil {
		filePath = *file.FilePath
	}
	duration := 0
	if file.Duration != nil {
		duration = *file.Duration
	}
	return domain.VideoInfoFile{
		FileID: file.FileID, UserID: file.UserID, VideoID: file.VideoID, FileName: file.FileName,
		FileIndex: file.FileIndex, FileSize: file.FileSize, FilePath: filePath, Duration: duration,
		DownloadStatus: domain.VideoDownloadStatusNone,
	}
}

func (r *VideoPostRepository) ListDownloadGenerationJobs(ctx context.Context, videoID string) ([]VideoDownloadGenerationJob, error) {
	var jobs []VideoDownloadGenerationJob
	err := r.db.WithContext(ctx).
		Table("video_info_file vf").
		Select(`
			vf.file_id,
			COALESCE(vf.file_name, '') AS file_name,
			COALESCE(vf.file_path, '') AS file_path,
			vf.video_id,
			COALESCE(vi.video_name, '') AS video_name,
			vi.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(vf.download_status, 0) AS download_status
		`).
		Joins("INNER JOIN video_info vi ON vi.video_id = vf.video_id").
		Joins("LEFT JOIN user_info ui ON ui.user_id = vi.user_id").
		Where("vf.video_id = ? AND COALESCE(vi.download_permission, 1) = ?", videoID, 1).
		Order("vf.file_index asc").
		Scan(&jobs).Error
	return jobs, err
}

func (r *VideoPostRepository) MarkVideoFileDownloadGenerating(ctx context.Context, fileID string) error {
	return r.db.WithContext(ctx).
		Table("video_info_file").
		Where("file_id = ?", fileID).
		Updates(map[string]any{
			"download_status":    domain.VideoDownloadStatusGenerating,
			"download_file_path": "",
		}).Error
}

func (r *VideoPostRepository) MarkVideoFileDownloadSuccess(ctx context.Context, fileID string, downloadFilePath string) error {
	return r.db.WithContext(ctx).
		Table("video_info_file").
		Where("file_id = ?", fileID).
		Updates(map[string]any{
			"download_status":    domain.VideoDownloadStatusSuccess,
			"download_file_path": downloadFilePath,
		}).Error
}

func (r *VideoPostRepository) MarkVideoFileDownloadFailed(ctx context.Context, fileID string) error {
	return r.db.WithContext(ctx).
		Table("video_info_file").
		Where("file_id = ?", fileID).
		Updates(map[string]any{
			"download_status":    domain.VideoDownloadStatusFailed,
			"download_file_path": "",
		}).Error
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

func (r *VideoPostRepository) FindTranscodeMessageByID(ctx context.Context, messageID string) (*domain.VideoTranscodeMessageRecord, error) {
	var message domain.VideoTranscodeMessageRecord
	if err := r.db.WithContext(ctx).Where("message_id = ?", messageID).Take(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
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

func createDynamicEventMessage(tx *gorm.DB, post domain.VideoInfoPost, eventType int) error {
	now := time.Now()
	eventType = normalizeDynamicEventType(eventType)
	event := domain.DynamicEvent{
		EventID:      dynamicEventID(eventType, post.VideoID),
		VideoID:      post.VideoID,
		AuthorUserID: post.UserID,
		DynamicTime:  post.LastUpdateTime,
		EventType:    eventType,
		CreateTime:   now,
		UpdateTime:   now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "video_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"author_user_id": event.AuthorUserID,
			"dynamic_time":   event.DynamicTime,
			"event_type":     event.EventType,
			"update_time":    now,
		}),
	}).Create(&event).Error; err != nil {
		return err
	}

	messageID, err := newDynamicFeedMessageID()
	if err != nil {
		return err
	}
	message := domain.DynamicFeedMessage{
		MessageID:    messageID,
		EventID:      event.EventID,
		VideoID:      event.VideoID,
		AuthorUserID: event.AuthorUserID,
		DynamicTime:  event.DynamicTime,
		EventType:    event.EventType,
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return tx.Create(&domain.DynamicFeedMessageRecord{
		MessageID:     messageID,
		EventID:       event.EventID,
		VideoID:       event.VideoID,
		AuthorUserID:  event.AuthorUserID,
		MessageStatus: domain.DynamicFeedMessageWaitPublish,
		Payload:       string(payload),
		NextRetryTime: &now,
		CreateTime:    now,
		UpdateTime:    now,
	}).Error
}

func dynamicEventID(eventType int, videoID string) string {
	if eventType == domain.DynamicEventTypeImage {
		return "image_" + strings.TrimSpace(videoID)
	}
	return "video_" + strings.TrimSpace(videoID)
}

func newDynamicFeedMessageID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "df" + hex.EncodeToString(buffer)[:30], nil
}
