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
	UserID          string
	FocusUserID     string
	PageSize        int
	LastUpdateTime  string
	LastContentType int
	LastContentID   string
	LastVideoID     string
	ReadFanCount    int64
}

const dynamicFeedContentSelectColumns = `
	feed.event_type,
	CASE WHEN feed.event_type = 2 THEN 1 ELSE 0 END AS content_type,
	feed.video_id AS content_id,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.video_cover ELSE vi.video_cover END, '') AS content_cover,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.video_name ELSE vi.video_name END, '') AS content_name,
	feed.video_id,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.video_cover ELSE vi.video_cover END, '') AS video_cover,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.video_name ELSE vi.video_name END, '') AS video_name,
	CASE WHEN feed.event_type = 2 THEN vip.user_id ELSE vi.user_id END AS user_id,
	COALESCE(ui.nick_name, '') AS nick_name,
	COALESCE(ui.avatar, '') AS avatar,
	DATE_FORMAT(CASE WHEN feed.event_type = 2 THEN vip.create_time ELSE vi.create_time END, '%Y-%m-%d %H:%i:%s') AS create_time,
	DATE_FORMAT(feed.dynamic_time, '%Y-%m-%d %H:%i:%s') AS last_update_time,
	CASE WHEN feed.event_type = 2 THEN vip.p_category_id ELSE vi.p_category_id END AS p_category_id,
	CASE WHEN feed.event_type = 2 THEN vip.category_id ELSE vi.category_id END AS category_id,
	CASE WHEN feed.event_type = 2 THEN vip.post_type ELSE vi.post_type END AS post_type,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.origin_info ELSE vi.origin_info END, '') AS origin_info,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.tags ELSE vi.tags END, '') AS tags,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.introduction ELSE vi.introduction END, '') AS introduction,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.interaction ELSE vi.interaction END, '') AS interaction,
	COALESCE(CASE WHEN feed.event_type = 2 THEN vip.download_permission ELSE vi.download_permission END, 1) AS download_permission,
	CASE WHEN feed.event_type = 2 THEN 0 ELSE COALESCE(vi.duration, 0) END AS duration,
	COALESCE(vi.play_count, 0) AS play_count,
	COALESCE(vi.like_count, 0) AS like_count,
	COALESCE(vi.danmu_count, 0) AS danmu_count,
	COALESCE(vi.comment_count, 0) AS comment_count,
	COALESCE(vi.coin_count, 0) AS coin_count,
	COALESCE(vi.collect_count, 0) AS collect_count,
	COALESCE(vi.recommend_type, 0) AS recommend_type
`

const dynamicFeedVideoDetailSelectColumns = `
	1 AS event_type,
	0 AS content_type,
	vi.video_id AS content_id,
	COALESCE(vi.video_cover, '') AS content_cover,
	COALESCE(vi.video_name, '') AS content_name,
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

const dynamicFeedImageDetailSelectColumns = `
	2 AS event_type,
	1 AS content_type,
	v.video_id AS content_id,
	COALESCE(v.video_cover, '') AS content_cover,
	COALESCE(v.video_name, '') AS content_name,
	v.video_id,
	COALESCE(v.video_cover, '') AS video_cover,
	COALESCE(v.video_name, '') AS video_name,
	v.user_id,
	COALESCE(ui.nick_name, '') AS nick_name,
	COALESCE(ui.avatar, '') AS avatar,
	DATE_FORMAT(v.create_time, '%Y-%m-%d %H:%i:%s') AS create_time,
	DATE_FORMAT(v.last_update_time, '%Y-%m-%d %H:%i:%s') AS last_update_time,
	v.p_category_id,
	v.category_id,
	v.post_type,
	COALESCE(v.origin_info, '') AS origin_info,
	COALESCE(v.tags, '') AS tags,
	COALESCE(v.introduction, '') AS introduction,
	COALESCE(v.interaction, '') AS interaction,
	COALESCE(v.download_permission, 1) AS download_permission,
	0 AS duration,
	0 AS play_count,
	0 AS like_count,
	0 AS danmu_count,
	0 AS comment_count,
	0 AS coin_count,
	0 AS collect_count,
	0 AS recommend_type
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
	query.LastContentID = strings.TrimSpace(query.LastContentID)
	if query.LastContentID == "" {
		query.LastContentID = strings.TrimSpace(query.LastVideoID)
	}
	lastEventType := dynamicEventTypeFromContentType(query.LastContentType)

	inboxSQL := `
		SELECT udf.video_id, udf.author_user_id, udf.dynamic_time, COALESCE(udf.event_type, 1) AS event_type
		FROM user_dynamic_feed udf
		WHERE udf.user_id = ?
	`
	inboxArgs := []any{query.UserID}
	if query.FocusUserID != "" {
		inboxSQL += " AND udf.author_user_id = ?"
		inboxArgs = append(inboxArgs, query.FocusUserID)
	}
	if query.LastUpdateTime != "" && query.LastContentID != "" {
		inboxSQL += " AND (DATE_FORMAT(udf.dynamic_time, '%Y-%m-%d %H:%i:%s') < ? OR (DATE_FORMAT(udf.dynamic_time, '%Y-%m-%d %H:%i:%s') = ? AND (COALESCE(udf.event_type, 1) < ? OR (COALESCE(udf.event_type, 1) = ? AND udf.video_id < ?))))"
		inboxArgs = append(inboxArgs, query.LastUpdateTime, query.LastUpdateTime, lastEventType, lastEventType, query.LastContentID)
	}

	readSQL := `
		SELECT de.video_id, de.author_user_id, de.dynamic_time, de.event_type
		FROM dynamic_event de
		INNER JOIN user_focus uf
			ON uf.focus_user_id = de.author_user_id
			AND uf.user_id = ?
			AND uf.focus_time <= de.dynamic_time
		LEFT JOIN user_dynamic_feed existing
			ON existing.user_id = ?
			AND existing.video_id = de.video_id
			AND COALESCE(existing.event_type, 1) = de.event_type
		WHERE existing.feed_id IS NULL
			AND de.event_type IN (?, ?)
			AND (
				SELECT COUNT(*)
				FROM user_focus fans
				WHERE fans.focus_user_id = de.author_user_id
			) >= ?
	`
	readArgs := []any{query.UserID, query.UserID, domain.DynamicEventTypeVideo, domain.DynamicEventTypeImage, query.ReadFanCount}
	if query.FocusUserID != "" {
		readSQL += " AND de.author_user_id = ?"
		readArgs = append(readArgs, query.FocusUserID)
	}
	if query.LastUpdateTime != "" && query.LastContentID != "" {
		readSQL += " AND (DATE_FORMAT(de.dynamic_time, '%Y-%m-%d %H:%i:%s') < ? OR (DATE_FORMAT(de.dynamic_time, '%Y-%m-%d %H:%i:%s') = ? AND (de.event_type < ? OR (de.event_type = ? AND de.video_id < ?))))"
		readArgs = append(readArgs, query.LastUpdateTime, query.LastUpdateTime, lastEventType, lastEventType, query.LastContentID)
	}

	sql := `
		SELECT ` + dynamicFeedContentSelectColumns + `
		FROM (` + inboxSQL + ` UNION ALL ` + readSQL + `) feed
		LEFT JOIN video_info vi
			ON feed.event_type = ?
			AND vi.video_id = feed.video_id
		LEFT JOIN video_info_post vip
			ON feed.event_type = ?
			AND vip.video_id = feed.video_id
			AND COALESCE(vip.content_type, 0) = ?
			AND vip.status = ?
		LEFT JOIN user_info ui
			ON ui.user_id = CASE WHEN feed.event_type = ? THEN vip.user_id ELSE vi.user_id END
		WHERE (feed.event_type = ? AND vi.video_id IS NOT NULL)
			OR (feed.event_type = ? AND vip.video_id IS NOT NULL)
		ORDER BY feed.dynamic_time DESC, feed.event_type DESC, feed.video_id DESC
		LIMIT ?
	`
	args := append(inboxArgs, readArgs...)
	args = append(args,
		domain.DynamicEventTypeVideo,
		domain.DynamicEventTypeImage,
		domain.ContentTypeImage,
		domain.VideoPostStatusApproved,
		domain.DynamicEventTypeImage,
		domain.DynamicEventTypeVideo,
		domain.DynamicEventTypeImage,
	)
	args = append(args, query.PageSize)

	err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *DynamicRepository) ListFeedContentDetailsByKeys(ctx context.Context, keys []domain.DynamicFeedContentKey) ([]domain.WebVideoItem, error) {
	if len(keys) == 0 {
		return []domain.WebVideoItem{}, nil
	}

	videoIDs, imageIDs := splitDynamicFeedContentKeys(keys)
	var list []domain.WebVideoItem
	if len(videoIDs) > 0 {
		var videos []domain.WebVideoItem
		err := r.db.WithContext(ctx).
			Table("video_info vi").
			Select(dynamicFeedVideoDetailSelectColumns).
			Joins("LEFT JOIN user_info ui ON vi.user_id = ui.user_id").
			Where("vi.video_id IN ?", videoIDs).
			Scan(&videos).Error
		if err != nil {
			return nil, err
		}
		list = append(list, videos...)
	}
	if len(imageIDs) > 0 {
		var images []domain.WebVideoItem
		err := r.db.WithContext(ctx).
			Table("video_info_post v").
			Select(dynamicFeedImageDetailSelectColumns).
			Joins("LEFT JOIN user_info ui ON v.user_id = ui.user_id").
			Where("v.video_id IN ?", imageIDs).
			Where("COALESCE(v.content_type, 0) = ?", domain.ContentTypeImage).
			Where("v.status = ?", domain.VideoPostStatusApproved).
			Scan(&images).Error
		if err != nil {
			return nil, err
		}
		list = append(list, images...)
	}
	if list == nil {
		list = []domain.WebVideoItem{}
	}
	return list, nil
}

func (r *DynamicRepository) UpsertFeedItems(ctx context.Context, userID string, list []domain.WebVideoItem) error {
	for _, item := range list {
		contentID := dynamicFeedItemContentID(item)
		eventType := dynamicEventTypeFromContentType(item.ContentType)
		if contentID == "" || item.UserID == "" || item.LastUpdateTime == "" {
			continue
		}
		if err := r.db.WithContext(ctx).Exec(`
			INSERT INTO user_dynamic_feed (
				feed_id, user_id, author_user_id, video_id, event_type, dynamic_time, push_time, create_time, update_time
			) VALUES (
				?, ?, ?, ?, ?, STR_TO_DATE(?, '%Y-%m-%d %H:%i:%s'), NOW(), NOW(), NOW()
			)
			ON DUPLICATE KEY UPDATE
				author_user_id = VALUES(author_user_id),
				event_type = VALUES(event_type),
				dynamic_time = VALUES(dynamic_time),
				update_time = NOW()
		`, dynamicFeedID(userID, eventType, contentID), userID, item.UserID, contentID, eventType, item.LastUpdateTime).Error; err != nil {
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
	feedIDContentPart := dynamicFeedIDContentPart(message.EventType, message.VideoID)
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO user_dynamic_feed (
			feed_id, user_id, author_user_id, video_id, event_type, dynamic_time, push_time, create_time, update_time
		)
		SELECT
			CONCAT(uf.user_id, '_', ?),
			uf.user_id,
			?,
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
			event_type = VALUES(event_type),
			dynamic_time = VALUES(dynamic_time),
			update_time = NOW()
	`, feedIDContentPart, message.AuthorUserID, message.VideoID, normalizeDynamicEventType(message.EventType), message.DynamicTime, message.AuthorUserID, message.DynamicTime)
	return result.RowsAffected, result.Error
}

func (r *DynamicRepository) UpsertFeedForActiveFollowers(ctx context.Context, message domain.DynamicFeedMessage, activeUserIDs []string) (int64, error) {
	if len(activeUserIDs) == 0 {
		return 0, nil
	}
	feedIDContentPart := dynamicFeedIDContentPart(message.EventType, message.VideoID)
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO user_dynamic_feed (
			feed_id, user_id, author_user_id, video_id, event_type, dynamic_time, push_time, create_time, update_time
		)
		SELECT
			CONCAT(uf.user_id, '_', ?),
			uf.user_id,
			?,
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
			event_type = VALUES(event_type),
			dynamic_time = VALUES(dynamic_time),
			update_time = NOW()
	`, feedIDContentPart, message.AuthorUserID, message.VideoID, normalizeDynamicEventType(message.EventType), message.DynamicTime, message.AuthorUserID, message.DynamicTime, activeUserIDs)
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

func splitDynamicFeedContentKeys(keys []domain.DynamicFeedContentKey) ([]string, []string) {
	videoIDs := make([]string, 0, len(keys))
	imageIDs := make([]string, 0, len(keys))
	seenVideos := make(map[string]struct{})
	seenImages := make(map[string]struct{})
	for _, key := range keys {
		contentID := strings.TrimSpace(key.ContentID)
		if contentID == "" {
			continue
		}
		if key.ContentType == domain.ContentTypeImage {
			if _, exists := seenImages[contentID]; !exists {
				imageIDs = append(imageIDs, contentID)
				seenImages[contentID] = struct{}{}
			}
			continue
		}
		if _, exists := seenVideos[contentID]; !exists {
			videoIDs = append(videoIDs, contentID)
			seenVideos[contentID] = struct{}{}
		}
	}
	return videoIDs, imageIDs
}

func dynamicFeedItemContentID(item domain.WebVideoItem) string {
	contentID := strings.TrimSpace(item.ContentID)
	if contentID != "" {
		return contentID
	}
	return strings.TrimSpace(item.VideoID)
}

func dynamicEventTypeFromContentType(contentType int) int {
	if contentType == domain.ContentTypeImage {
		return domain.DynamicEventTypeImage
	}
	return domain.DynamicEventTypeVideo
}

func dynamicContentTypeFromEventType(eventType int) int {
	if eventType == domain.DynamicEventTypeImage {
		return domain.ContentTypeImage
	}
	return domain.ContentTypeVideo
}

func normalizeDynamicEventType(eventType int) int {
	if eventType == domain.DynamicEventTypeImage {
		return domain.DynamicEventTypeImage
	}
	return domain.DynamicEventTypeVideo
}

func dynamicFeedID(userID string, eventType int, contentID string) string {
	return strings.TrimSpace(userID) + "_" + dynamicFeedIDContentPart(eventType, contentID)
}

func dynamicFeedIDContentPart(eventType int, contentID string) string {
	contentID = strings.TrimSpace(contentID)
	if eventType == domain.DynamicEventTypeImage {
		return "image_" + contentID
	}
	return contentID
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
