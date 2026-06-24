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
