package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type VideoRepository struct {
	db *gorm.DB
}

type WebVideoListQuery struct {
	PageNo        int
	PageSize      int
	PCategoryID   int
	CategoryID    int
	RecommendType int
}

func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

const webVideoSelectColumns = `
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

const videoHotScoreExpression = `
	COALESCE(vi.play_count, 0) * 1 +
	COALESCE(vi.like_count, 0) * 5 +
	COALESCE(vi.collect_count, 0) * 5 +
	COALESCE(vi.coin_count, 0) * 6 +
	COALESCE(vi.comment_count, 0) * 8
`

func (r *VideoRepository) ListVideoFiles(ctx context.Context, videoID string) ([]domain.VideoInfoFile, error) {
	var files []domain.VideoInfoFile
	err := r.db.WithContext(ctx).
		Table("video_info_file").
		Select(`
			file_id,
			user_id,
			video_id,
			COALESCE(file_name, '') AS file_name,
			file_index,
			file_size,
			COALESCE(file_path, '') AS file_path,
			COALESCE(duration, 0) AS duration,
			COALESCE(download_status, 0) AS download_status,
			COALESCE(download_file_path, '') AS download_file_path
		`).
		Where("video_id = ?", videoID).
		Order("file_index asc").
		Scan(&files).Error
	return files, err
}

func (r *VideoRepository) FindVideoFileByFileID(ctx context.Context, fileID string) (*domain.VideoInfoFile, error) {
	var file domain.VideoInfoFile
	err := r.db.WithContext(ctx).
		Table("video_info_file").
		Select(`
			file_id,
			user_id,
			video_id,
			COALESCE(file_name, '') AS file_name,
			file_index,
			file_size,
			COALESCE(file_path, '') AS file_path,
			COALESCE(duration, 0) AS duration,
			COALESCE(download_status, 0) AS download_status,
			COALESCE(download_file_path, '') AS download_file_path
		`).
		Where("file_id = ?", fileID).
		Take(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *VideoRepository) FindDownloadVideoFileByFileID(ctx context.Context, fileID string) (*domain.DownloadVideoFile, error) {
	var file domain.DownloadVideoFile
	err := r.db.WithContext(ctx).
		Table("video_info_file vf").
		Select(`
			vf.file_id,
			COALESCE(vf.file_name, '') AS file_name,
			vf.video_id,
			COALESCE(vi.video_name, '') AS video_name,
			COALESCE(vi.download_permission, 1) AS download_permission,
			COALESCE(vf.download_status, 0) AS download_status,
			COALESCE(vf.download_file_path, '') AS download_file_path
		`).
		Joins("INNER JOIN video_info vi ON vi.video_id = vf.video_id").
		Where("vf.file_id = ?", fileID).
		Take(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *VideoRepository) FindWebVideoByID(ctx context.Context, videoID string) (*domain.WebVideoItem, error) {
	var video domain.WebVideoItem
	err := r.db.WithContext(ctx).
		Table("video_info vi").
		Select(webVideoSelectColumns).
		Joins("LEFT JOIN user_info ui ON vi.user_id = ui.user_id").
		Where("vi.video_id = ?", videoID).
		Take(&video).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *VideoRepository) ListWebVideoByIDs(ctx context.Context, videoIDs []string) ([]domain.WebVideoItem, error) {
	if len(videoIDs) == 0 {
		return []domain.WebVideoItem{}, nil
	}
	if r == nil || r.db == nil {
		return []domain.WebVideoItem{}, nil
	}

	var list []domain.WebVideoItem
	err := r.db.WithContext(ctx).
		Table("video_info vi").
		Select(webVideoSelectColumns).
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

func (r *VideoRepository) IncrementVideoPlayCount(ctx context.Context, videoID string, delta int) error {
	if r == nil || r.db == nil || videoID == "" || delta == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Table("video_info").
		Where("video_id = ?", videoID).
		Update("play_count", gorm.Expr("GREATEST(COALESCE(play_count, 0) + ?, 0)", delta)).Error
}

func (r *VideoRepository) ListVideoHotMetricSnapshots(ctx context.Context, offset int, limit int) ([]domain.VideoHotMetrics, error) {
	if r == nil || r.db == nil {
		return []domain.VideoHotMetrics{}, nil
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 200
	}

	var list []domain.VideoHotMetrics
	err := r.db.WithContext(ctx).
		Table("video_info vi").
		Select(`
			vi.video_id,
			COALESCE(vi.play_count, 0) AS play_count,
			COALESCE(vi.like_count, 0) AS like_count,
			COALESCE(vi.collect_count, 0) AS collect_count,
			COALESCE(vi.coin_count, 0) AS coin_count,
			COALESCE(vi.comment_count, 0) AS comment_count
		`).
		Order("vi.video_id asc").
		Offset(offset).
		Limit(limit).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.VideoHotMetrics{}
	}
	return list, nil
}

func (r *VideoRepository) ListWebHotVideoByDBPage(ctx context.Context, pageNo int, pageSize int) ([]domain.WebVideoItem, int64, error) {
	if r == nil || r.db == nil {
		return []domain.WebVideoItem{}, 0, nil
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var totalCount int64
	if err := r.db.WithContext(ctx).Table("video_info vi").Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (pageNo - 1) * pageSize
	var list []domain.WebVideoItem
	err := r.db.WithContext(ctx).
		Table("video_info vi").
		Select(webVideoSelectColumns).
		Joins("LEFT JOIN user_info ui ON vi.user_id = ui.user_id").
		Order(videoHotScoreExpression + " desc").
		Order("vi.last_update_time desc").
		Order("vi.video_id asc").
		Offset(offset).
		Limit(pageSize).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []domain.WebVideoItem{}
	}
	return list, totalCount, nil
}

func (r *VideoRepository) FindVideoSearchDocumentByID(ctx context.Context, videoID string) (*domain.VideoSearchDocument, error) {
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

func (r *VideoRepository) ListVideoSearchDocuments(ctx context.Context, offset int, limit int) ([]domain.VideoSearchDocument, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 200
	}

	var list []domain.VideoSearchDocument
	err := r.db.WithContext(ctx).
		Table("video_info").
		Select(videoSearchDocumentSelectColumns).
		Order("create_time asc").
		Offset(offset).
		Limit(limit).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.VideoSearchDocument{}
	}
	return list, nil
}

func (r *VideoRepository) ListUserVideoActions(ctx context.Context, videoID string, userID string) ([]domain.UserActionItem, error) {
	if userID == "" {
		return []domain.UserActionItem{}, nil
	}

	var actions []domain.UserActionItem
	err := r.db.WithContext(ctx).
		Table("user_action").
		Select(`
			action_id,
			video_id,
			comment_id,
			action_type,
			action_count,
			user_id,
			COALESCE(DATE_FORMAT(action_time, '%Y-%m-%d %H:%i:%s'), '') AS action_time
		`).
		Where("video_id = ? AND user_id = ? AND comment_id = ? AND action_type IN ?", videoID, userID, 0, []int{2, 3, 4}).
		Order("action_id desc").
		Scan(&actions).Error
	if err != nil {
		return nil, err
	}
	if actions == nil {
		actions = []domain.UserActionItem{}
	}
	return actions, nil
}

const videoSearchDocumentSelectColumns = `
	video_id,
	user_id,
	COALESCE(video_cover, '') AS video_cover,
	COALESCE(video_name, '') AS video_name,
	COALESCE(tags, '') AS tags,
	COALESCE(play_count, 0) AS play_count,
	COALESCE(danmu_count, 0) AS danmu_count,
	COALESCE(collect_count, 0) AS collect_count,
	DATE_FORMAT(create_time, '%Y-%m-%d %H:%i:%s') AS create_time
`

func (r *VideoRepository) ListWebVideoByPage(ctx context.Context, query WebVideoListQuery) ([]domain.WebVideoItem, int64, error) {
	baseQuery := r.applyWebVideoListFilter(r.db.WithContext(ctx).Table("video_info vi"), query)

	var totalCount int64
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.WebVideoItem
	offset := (query.PageNo - 1) * query.PageSize
	err := r.applyWebVideoListFilter(r.db.WithContext(ctx).Table("video_info vi"), query).
		Select(webVideoSelectColumns).
		Joins("LEFT JOIN user_info ui ON vi.user_id = ui.user_id").
		Order("vi.create_time desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, totalCount, nil
}

func (r *VideoRepository) ListWebVideo(ctx context.Context, query WebVideoListQuery) ([]domain.WebVideoItem, error) {
	var list []domain.WebVideoItem
	err := r.applyWebVideoListFilter(r.db.WithContext(ctx).Table("video_info vi"), query).
		Select(webVideoSelectColumns).
		Joins("LEFT JOIN user_info ui ON vi.user_id = ui.user_id").
		Order("vi.create_time desc").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *VideoRepository) ListUserAllVideo(ctx context.Context, userID string) ([]domain.UcenterAllVideoItem, error) {
	var list []domain.UcenterAllVideoItem
	err := r.db.WithContext(ctx).
		Table("video_info").
		Select(`
			video_id,
			COALESCE(video_cover, '') AS video_cover,
			COALESCE(video_name, '') AS video_name,
			COALESCE(DATE_FORMAT(create_time, '%Y-%m-%d %H:%i:%s'), '') AS create_time,
			COALESCE(danmu_count, 0) AS danmu_count,
			COALESCE(comment_count, 0) AS comment_count
		`).
		Where("user_id = ?", userID).
		Order("create_time desc").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []domain.UcenterAllVideoItem{}
	}
	return list, nil
}

func (r *VideoRepository) applyWebVideoListFilter(db *gorm.DB, query WebVideoListQuery) *gorm.DB {
	db = db.Where("vi.recommend_type = ?", query.RecommendType)
	if query.PCategoryID > 0 {
		db = db.Where("vi.p_category_id = ?", query.PCategoryID)
	}
	if query.CategoryID > 0 {
		db = db.Where("vi.category_id = ?", query.CategoryID)
	}
	return db
}
