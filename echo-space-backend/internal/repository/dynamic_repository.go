package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const dynamicMessageErrorMaxLength = 500

type DynamicRepository struct {
	db *gorm.DB
}

type DynamicFeedQuery struct {
	UserID         string
	FocusUserID    string
	PageSize       int
	LastUpdateTime string
	LastVideoID    string
	ReadFanCount   int64
}

const dynamicFeedVideoSelectColumns = `
	vi.video_id,
	COALESCE(vi.video_cover, '') AS video_cover,
	COALESCE(vi.video_name, '') AS video_name,
	vi.user_id,
	COALESCE(ui.nick_name, '') AS nick_name,
	COALESCE(ui.avatar, '') AS avatar,
	DATE_FORMAT(vi.create_time, '%Y-%m-%d %H:%i:%s') AS create_time,
	DATE_FORMAT(feed.dynamic_time, '%Y-%m-%d %H:%i:%s') AS last_update_time,
	vi.p_category_id,
	vi.category_id,
	vi.post_type,
	COALESCE(vi.origin_info, '') AS origin_info,
	COALESCE(vi.tags, '') AS tags,
	COALESCE(vi.introduction, '') AS introduction,
	COALESCE(vi.interaction, '') AS interaction,
	COALESCE(vi.download_permission, 1) AS download_permission,
	COALESCE(vi.duration, 0) AS duration,
	COALESCE(vi.play_count, 0) AS play_count,
	COALESCE(vi.like_count, 0) AS like_count,
	COALESCE(vi.danmu_count, 0) AS danmu_count,
	COALESCE(vi.comment_count, 0) AS comment_count,
	COALESCE(vi.coin_count, 0) AS coin_count,
	COALESCE(vi.collect_count, 0) AS collect_count,
	COALESCE(vi.recommend_type, 0) AS recommend_type
`

const dynamicFeedVideoDetailSelectColumns = `
	vi.video_id,
	COALESCE(vi.video_cover, '') AS video_cover,
	COALESCE(vi.video_name, '') AS video_name,
	vi.user_id,
	COALESCE(ui.nick_name, '') AS nick_name,
	COALESCE(ui.avatar, '') AS avatar,
	DATE_FORMAT(vi.create_time, '%Y-%m-%d %H:%i:%s') AS create_time,
	DATE_FORMAT(vi.last_update_time, '%Y-%m-%d %H:%i:%s') AS last_update_time,
	vi.p_category_id,
	vi.category_id,
	vi.post_type,
	COALESCE(vi.origin_info, '') AS origin_info,
	COALESCE(vi.tags, '') AS tags,
	COALESCE(vi.introduction, '') AS introduction,
	COALESCE(vi.interaction, '') AS interaction,
	COALESCE(vi.download_permission, 1) AS download_permission,
	COALESCE(vi.duration, 0) AS duration,
	COALESCE(vi.play_count, 0) AS play_count,
	COALESCE(vi.like_count, 0) AS like_count,
	COALESCE(vi.danmu_count, 0) AS danmu_count,
	COALESCE(vi.comment_count, 0) AS comment_count,
	COALESCE(vi.coin_count, 0) AS coin_count,
	COALESCE(vi.collect_count, 0) AS collect_count,
	COALESCE(vi.recommend_type, 0) AS recommend_type
`

func NewDynamicRepository(db *gorm.DB) *DynamicRepository {
	return &DynamicRepository{db: db}
}

func (r *DynamicRepository) FindCurrentUserInfo(ctx context.Context, userID string) (*domain.DynamicCurrentUserInfo, error) {
	var info domain.DynamicCurrentUserInfo
	err := r.db.WithContext(ctx).
		Table("user_info ui").
		Select(`
			ui.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(ui.avatar, '') AS avatar,
			(
				SELECT COUNT(*)
				FROM user_focus uf
				WHERE uf.user_id = ui.user_id
			) AS focus_count,
			(
				SELECT COUNT(*)
				FROM user_focus uf
				WHERE uf.focus_user_id = ui.user_id
			) AS fans_count,
			(
				SELECT COUNT(*)
				FROM video_info vi
				WHERE vi.user_id = ui.user_id
			) AS dynamic_count
		`).
		Where("ui.user_id = ?", userID).
		Take(&info).Error
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (r *DynamicRepository) ListFollowUsers(ctx context.Context, userID string) ([]domain.DynamicFollowUserItem, error) {
	var list []domain.DynamicFollowUserItem
	err := r.db.WithContext(ctx).
		Table("user_focus uf").
		Select(`
			ui.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(ui.avatar, '') AS avatar,
			COALESCE(ui.person_introduction, '') AS person_introduction,
			DATE_FORMAT(uf.focus_time, '%Y-%m-%d %H:%i:%s') AS focus_time
		`).
		Joins("INNER JOIN user_info ui ON ui.user_id = uf.focus_user_id").
		Where("uf.user_id = ?", userID).
		Order("uf.focus_time desc").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *DynamicRepository) ListFeedByCursor(ctx context.Context, query DynamicFeedQuery) ([]domain.WebVideoItem, error) {
	var list []domain.WebVideoItem
	if query.ReadFanCount <= 0 {
		query.ReadFanCount = 1000
	}

	inboxSQL := `
		SELECT udf.video_id, udf.author_user_id, udf.dynamic_time
		FROM user_dynamic_feed udf
		WHERE udf.user_id = ?
	`
	inboxArgs := []any{query.UserID}
	if query.FocusUserID != "" {
		inboxSQL += " AND udf.author_user_id = ?"
		inboxArgs = append(inboxArgs, query.FocusUserID)
	}
	if query.LastUpdateTime != "" && query.LastVideoID != "" {
		inboxSQL += " AND (DATE_FORMAT(udf.dynamic_time, '%Y-%m-%d %H:%i:%s') < ? OR (DATE_FORMAT(udf.dynamic_time, '%Y-%m-%d %H:%i:%s') = ? AND udf.video_id < ?))"
		inboxArgs = append(inboxArgs, query.LastUpdateTime, query.LastUpdateTime, query.LastVideoID)
	}

	readSQL := `
		SELECT de.video_id, de.author_user_id, de.dynamic_time
		FROM dynamic_event de
		INNER JOIN user_focus uf
			ON uf.focus_user_id = de.author_user_id
			AND uf.user_id = ?
			AND uf.focus_time <= de.dynamic_time
		LEFT JOIN user_dynamic_feed existing
			ON existing.user_id = ?
			AND existing.video_id = de.video_id
		WHERE existing.feed_id IS NULL
			AND de.event_type = ?
			AND (
				SELECT COUNT(*)
				FROM user_focus fans
				WHERE fans.focus_user_id = de.author_user_id
			) >= ?
	`
	readArgs := []any{query.UserID, query.UserID, domain.DynamicEventTypeVideo, query.ReadFanCount}
	if query.FocusUserID != "" {
		readSQL += " AND de.author_user_id = ?"
		readArgs = append(readArgs, query.FocusUserID)
	}
	if query.LastUpdateTime != "" && query.LastVideoID != "" {
		readSQL += " AND (DATE_FORMAT(de.dynamic_time, '%Y-%m-%d %H:%i:%s') < ? OR (DATE_FORMAT(de.dynamic_time, '%Y-%m-%d %H:%i:%s') = ? AND de.video_id < ?))"
		readArgs = append(readArgs, query.LastUpdateTime, query.LastUpdateTime, query.LastVideoID)
	}

	sql := `
		SELECT ` + dynamicFeedVideoSelectColumns + `
		FROM (` + inboxSQL + ` UNION ALL ` + readSQL + `) feed
		INNER JOIN video_info vi ON vi.video_id = feed.video_id
		LEFT JOIN user_info ui ON vi.user_id = ui.user_id
		ORDER BY feed.dynamic_time DESC, feed.video_id DESC
		LIMIT ?
	`
	args := append(inboxArgs, readArgs...)
	args = append(args, query.PageSize)

	err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *DynamicRepository) ListFeedVideoDetailsByIDs(ctx context.Context, videoIDs []string) ([]domain.WebVideoItem, error) {
	if len(videoIDs) == 0 {
		return []domain.WebVideoItem{}, nil
	}

	var list []domain.WebVideoItem
	err := r.db.WithContext(ctx).
		Table("video_info vi").
		Select(dynamicFeedVideoDetailSelectColumns).
		Joins("LEFT JOIN user_info ui ON vi.user_id = ui.user_id").
		Where("vi.video_id IN ?", videoIDs).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.WebVideoItem{}
	}
	return list, nil
}

func (r *DynamicRepository) UpsertFeedItems(ctx context.Context, userID string, list []domain.WebVideoItem) error {
	for _, item := range list {
		if item.VideoID == "" || item.UserID == "" || item.LastUpdateTime == "" {
			continue
		}
		if err := r.db.WithContext(ctx).Exec(`
			INSERT INTO user_dynamic_feed (
				feed_id, user_id, author_user_id, video_id, dynamic_time, push_time, create_time, update_time
			) VALUES (
				?, ?, ?, ?, STR_TO_DATE(?, '%Y-%m-%d %H:%i:%s'), NOW(), NOW(), NOW()
			)
			ON DUPLICATE KEY UPDATE
				author_user_id = VALUES(author_user_id),
				dynamic_time = VALUES(dynamic_time),
				update_time = NOW()
		`, dynamicFeedID(userID, item.VideoID), userID, item.UserID, item.VideoID, item.LastUpdateTime).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *DynamicRepository) CountFans(ctx context.Context, authorUserID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_focus").
		Where("focus_user_id = ?", authorUserID).
		Count(&count).Error
	return count, err
}

func (r *DynamicRepository) UpsertFeedForAllFollowers(ctx context.Context, message domain.DynamicFeedMessage) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO user_dynamic_feed (
			feed_id, user_id, author_user_id, video_id, dynamic_time, push_time, create_time, update_time
		)
		SELECT
			CONCAT(uf.user_id, '_', ?),
			uf.user_id,
			?,
			?,
			?,
			NOW(),
			NOW(),
			NOW()
		FROM user_focus uf
		WHERE uf.focus_user_id = ?
			AND uf.focus_time <= ?
		ON DUPLICATE KEY UPDATE
			author_user_id = VALUES(author_user_id),
			dynamic_time = VALUES(dynamic_time),
			update_time = NOW()
	`, message.VideoID, message.AuthorUserID, message.VideoID, message.DynamicTime, message.AuthorUserID, message.DynamicTime)
	return result.RowsAffected, result.Error
}

func (r *DynamicRepository) UpsertFeedForActiveFollowers(ctx context.Context, message domain.DynamicFeedMessage, activeUserIDs []string) (int64, error) {
	if len(activeUserIDs) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO user_dynamic_feed (
			feed_id, user_id, author_user_id, video_id, dynamic_time, push_time, create_time, update_time
		)
		SELECT
			CONCAT(uf.user_id, '_', ?),
			uf.user_id,
			?,
			?,
			?,
			NOW(),
			NOW(),
			NOW()
		FROM user_focus uf
		WHERE uf.focus_user_id = ?
			AND uf.focus_time <= ?
			AND uf.user_id IN ?
		ON DUPLICATE KEY UPDATE
			author_user_id = VALUES(author_user_id),
			dynamic_time = VALUES(dynamic_time),
			update_time = NOW()
	`, message.VideoID, message.AuthorUserID, message.VideoID, message.DynamicTime, message.AuthorUserID, message.DynamicTime, activeUserIDs)
	return result.RowsAffected, result.Error
}

func (r *DynamicRepository) ListFanoutUserIDs(ctx context.Context, authorUserID string, dynamicTime time.Time, candidateUserIDs []string) ([]string, error) {
	authorUserID = strings.TrimSpace(authorUserID)
	if authorUserID == "" {
		return []string{}, nil
	}
	if candidateUserIDs != nil && len(candidateUserIDs) == 0 {
		return []string{}, nil
	}

	query := r.db.WithContext(ctx).
		Table("user_focus").
		Select("user_id").
		Where("focus_user_id = ?", authorUserID).
		Where("focus_time <= ?", dynamicTime)
	if candidateUserIDs != nil {
		query = query.Where("user_id IN ?", candidateUserIDs)
	}

	var userIDs []string
	if err := query.Scan(&userIDs).Error; err != nil {
		return nil, err
	}
	if userIDs == nil {
		userIDs = []string{}
	}
	return userIDs, nil
}

func (r *DynamicRepository) ListDynamicFeedMessagesForPublish(ctx context.Context, limit int, publishedBefore time.Time) ([]domain.DynamicFeedMessageRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	now := time.Now()
	var list []domain.DynamicFeedMessageRecord
	err := r.db.WithContext(ctx).
		Where(`
			(message_status IN ? AND (next_retry_time IS NULL OR next_retry_time <= ?))
			OR (message_status = ? AND update_time <= ?)
			OR (message_status = ? AND locked_until <= ?)
		`, []int{domain.DynamicFeedMessageWaitPublish, domain.DynamicFeedMessageRetryWait}, now,
			domain.DynamicFeedMessagePublished, publishedBefore,
			domain.DynamicFeedMessageProcessing, now).
		Order("create_time asc").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (r *DynamicRepository) MarkDynamicFeedMessagePublished(ctx context.Context, messageID string, publishedBefore time.Time) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.DynamicFeedMessageRecord{}).
		Where("message_id = ?", messageID).
		Where(`
			message_status IN ?
			OR (message_status = ? AND update_time <= ?)
			OR (message_status = ? AND locked_until <= ?)
		`, []int{domain.DynamicFeedMessageWaitPublish, domain.DynamicFeedMessageRetryWait, domain.DynamicFeedMessagePublished},
			domain.DynamicFeedMessagePublished, publishedBefore,
			domain.DynamicFeedMessageProcessing, now).
		Updates(map[string]any{
			"message_status":  domain.DynamicFeedMessagePublished,
			"next_retry_time": nil,
			"locked_until":    nil,
			"lock_token":      "",
			"last_error":      "",
		}).Error
}

func (r *DynamicRepository) DelayDynamicFeedMessagePublish(ctx context.Context, messageID string, nextRetry time.Time, cause error) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.DynamicFeedMessageRecord{}).
		Where("message_id = ?", messageID).
		Where("message_status IN ? OR (message_status = ? AND locked_until <= ?)",
			[]int{domain.DynamicFeedMessageWaitPublish, domain.DynamicFeedMessageRetryWait, domain.DynamicFeedMessagePublished},
			domain.DynamicFeedMessageProcessing, now).
		Updates(map[string]any{
			"message_status":  domain.DynamicFeedMessageWaitPublish,
			"next_retry_time": nextRetry,
			"locked_until":    nil,
			"lock_token":      "",
			"last_error":      trimDynamicMessageError(cause),
		}).Error
}

func (r *DynamicRepository) ClaimDynamicFeedMessage(ctx context.Context, messageID string, lockToken string, leaseUntil time.Time) (*domain.DynamicFeedMessageRecord, bool, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&domain.DynamicFeedMessageRecord{}).
		Where("message_id = ?", messageID).
		Where(`
			(message_status IN ? AND (next_retry_time IS NULL OR next_retry_time <= ?))
			OR (message_status = ? AND locked_until <= ?)
		`, []int{domain.DynamicFeedMessageWaitPublish, domain.DynamicFeedMessagePublished, domain.DynamicFeedMessageRetryWait}, now,
			domain.DynamicFeedMessageProcessing, now).
		Updates(map[string]any{
			"message_status": domain.DynamicFeedMessageProcessing,
			"locked_until":   leaseUntil,
			"lock_token":     lockToken,
		})
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}

	var message domain.DynamicFeedMessageRecord
	if err := r.db.WithContext(ctx).Where("message_id = ?", messageID).Take(&message).Error; err != nil {
		return nil, false, err
	}
	return &message, true, nil
}

func (r *DynamicRepository) CompleteDynamicFeedMessage(ctx context.Context, messageID string, lockToken string) error {
	result := r.db.WithContext(ctx).Model(&domain.DynamicFeedMessageRecord{}).
		Where("message_id = ? AND message_status = ? AND lock_token = ?", messageID, domain.DynamicFeedMessageProcessing, lockToken).
		Updates(map[string]any{
			"message_status":  domain.DynamicFeedMessageSuccess,
			"next_retry_time": nil,
			"locked_until":    nil,
			"lock_token":      "",
			"last_error":      "",
		})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *DynamicRepository) RetryOrFailDynamicFeedMessage(ctx context.Context, messageID string, lockToken string, maxRetries int, nextRetry time.Time, cause error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var message domain.DynamicFeedMessageRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("message_id = ?", messageID).Take(&message).Error; err != nil {
			return err
		}
		if message.MessageStatus == domain.DynamicFeedMessageSuccess || message.MessageStatus == domain.DynamicFeedMessageDead {
			return nil
		}
		if lockToken != "" && (message.MessageStatus != domain.DynamicFeedMessageProcessing || message.LockToken != lockToken) {
			return nil
		}

		retryCount := message.RetryCount + 1
		status := domain.DynamicFeedMessageRetryWait
		var retryAt any = nextRetry
		if retryCount >= maxRetries {
			status = domain.DynamicFeedMessageDead
			retryAt = nil
		}
		return tx.Model(&domain.DynamicFeedMessageRecord{}).
			Where("message_id = ?", messageID).
			Updates(map[string]any{
				"message_status":  status,
				"retry_count":     retryCount,
				"next_retry_time": retryAt,
				"locked_until":    nil,
				"lock_token":      "",
				"last_error":      trimDynamicMessageError(cause),
			}).Error
	})
}

func dynamicFeedID(userID string, videoID string) string {
	return strings.TrimSpace(userID) + "_" + strings.TrimSpace(videoID)
}

func trimDynamicMessageError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > dynamicMessageErrorMaxLength {
		return message[:dynamicMessageErrorMaxLength]
	}
	return message
}
