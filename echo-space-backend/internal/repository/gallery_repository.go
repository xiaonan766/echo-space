package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type GalleryRepository struct {
	db *gorm.DB
}

func (r *GalleryRepository) ListApprovedVectorSourcesByImageID(ctx context.Context, imageID string) ([]domain.GalleryVectorSource, error) {
	var list []domain.GalleryVectorSource
	err := applyApprovedGalleryImageFilter(r.db.WithContext(ctx).Table("video_info_post v")).
		Select(`f.file_id, v.video_id AS image_id, f.file_path AS source_name, UNIX_TIMESTAMP(v.last_update_time) AS content_version`).
		Joins("JOIN video_info_file_post f ON f.video_id = v.video_id").
		Where("v.video_id = ?", imageID).
		Where("f.update_type <> ?", domain.VideoFileUpdateDeletePending).
		Where("f.transfer_result = ?", domain.VideoFileTransferSuccess).
		Where("COALESCE(f.file_path, '') <> ''").
		Order("f.file_index asc").
		Scan(&list).Error
	return list, err
}

func (r *GalleryRepository) ListApprovedImageIDs(ctx context.Context) ([]string, error) {
	var imageIDs []string
	err := applyApprovedGalleryImageFilter(r.db.WithContext(ctx).Table("video_info_post v")).
		Pluck("v.video_id", &imageIDs).Error
	return imageIDs, err
}

func (r *GalleryRepository) ListApprovedImagesByIDs(ctx context.Context, imageIDs []string) ([]domain.GalleryImageItem, error) {
	if len(imageIDs) == 0 {
		return []domain.GalleryImageItem{}, nil
	}
	var list []domain.GalleryImageItem
	err := applyApprovedGalleryImageFilter(r.db.WithContext(ctx).Table("video_info_post v")).
		Select(galleryImageSelectColumns).
		Joins("LEFT JOIN user_info u ON u.user_id = v.user_id").
		Where("v.video_id IN ?", imageIDs).
		Scan(&list).Error
	return list, err
}

func (r *GalleryRepository) IsApprovedImageFile(ctx context.Context, imageID, fileID string) (string, bool, error) {
	var sourceName string
	err := applyApprovedGalleryImageFilter(r.db.WithContext(ctx).Table("video_info_post v")).
		Select("COALESCE(f.file_path, '')").
		Joins("JOIN video_info_file_post f ON f.video_id = v.video_id").
		Where("v.video_id = ? AND f.file_id = ?", strings.TrimSpace(imageID), strings.TrimSpace(fileID)).
		Where("f.update_type <> ?", domain.VideoFileUpdateDeletePending).
		Where("f.transfer_result = ?", domain.VideoFileTransferSuccess).
		Where("COALESCE(f.file_path, '') <> ''").
		Scan(&sourceName).Error
	if err != nil {
		return "", false, err
	}
	return sourceName, sourceName != "", nil
}

type GalleryImageListQuery struct {
	PageNo   int
	PageSize int
}

func NewGalleryRepository(db *gorm.DB) *GalleryRepository {
	return &GalleryRepository{db: db}
}

const galleryImageSelectColumns = `
	v.video_id AS image_id,
	COALESCE(v.video_cover, '') AS image_cover,
	COALESCE(v.video_name, '') AS image_name,
	v.user_id,
	COALESCE(u.nick_name, '') AS nick_name,
	COALESCE(u.avatar, '') AS avatar,
	DATE_FORMAT(v.create_time, '%Y-%m-%d %H:%i:%s') AS create_time,
	DATE_FORMAT(v.last_update_time, '%Y-%m-%d %H:%i:%s') AS last_update_time
`

const galleryImageDetailSelectColumns = galleryImageSelectColumns + `,
	v.p_category_id,
	v.category_id,
	v.post_type,
	COALESCE(v.origin_info, '') AS origin_info,
	COALESCE(v.tags, '') AS tags,
	COALESCE(v.introduction, '') AS introduction
`

func (r *GalleryRepository) ListApprovedImagesByPage(ctx context.Context, query GalleryImageListQuery) ([]domain.GalleryImageItem, int64, error) {
	countQuery := applyApprovedGalleryImageFilter(r.db.WithContext(ctx).Table("video_info_post v"))
	var totalCount int64
	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.PageNo - 1) * query.PageSize
	var list []domain.GalleryImageItem
	err := applyApprovedGalleryImageFilter(r.db.WithContext(ctx).Table("video_info_post v")).
		Select(galleryImageSelectColumns).
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

func (r *GalleryRepository) FindApprovedImageByID(ctx context.Context, imageID string) (*domain.GalleryImageInfo, error) {
	var imageInfo domain.GalleryImageInfo
	err := applyApprovedGalleryImageFilter(r.db.WithContext(ctx).Table("video_info_post v")).
		Select(galleryImageDetailSelectColumns).
		Joins("LEFT JOIN user_info u ON u.user_id = v.user_id").
		Where("v.video_id = ?", imageID).
		Take(&imageInfo).Error
	if err != nil {
		return nil, err
	}
	return &imageInfo, nil
}

func (r *GalleryRepository) ListApprovedImageFiles(ctx context.Context, imageID string) ([]domain.GalleryImageFile, error) {
	var files []domain.GalleryImageFile
	err := r.db.WithContext(ctx).
		Table("video_info_file_post").
		Select(`
			file_id,
			COALESCE(file_name, '') AS file_name,
			COALESCE(file_path, '') AS source_name,
			file_index
		`).
		Scopes(func(db *gorm.DB) *gorm.DB {
			return applyApprovedGalleryImageFileFilter(db, imageID)
		}).
		Order("file_index asc").
		Scan(&files).Error
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = []domain.GalleryImageFile{}
	}
	return files, nil
}

func applyApprovedGalleryImageFilter(db *gorm.DB) *gorm.DB {
	return db.Where("COALESCE(v.content_type, 0) = ?", domain.ContentTypeImage).
		Where("v.status = ?", domain.VideoPostStatusApproved)
}

func applyApprovedGalleryImageFileFilter(db *gorm.DB, imageID string) *gorm.DB {
	return db.Where("video_id = ?", imageID).
		Where("update_type <> ?", domain.VideoFileUpdateDeletePending).
		Where("transfer_result = ?", domain.VideoFileTransferSuccess).
		Where("COALESCE(file_path, '') <> ''")
}
