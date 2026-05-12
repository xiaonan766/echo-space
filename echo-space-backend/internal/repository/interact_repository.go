package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

type InteractRepository struct {
	db *gorm.DB
}

type InteractListQuery struct {
	PageNo         int
	PageSize       int
	VideoNameFuzzy string
}

func NewInteractRepository(db *gorm.DB) *InteractRepository {
	return &InteractRepository{
		db: db,
	}
}

func (r *InteractRepository) ListCommentByPage(ctx context.Context, query InteractListQuery) ([]domain.AdminCommentItem, int64, error) {
	var totalCount int64
	countDB := r.applyVideoNameFilter(r.db.WithContext(ctx).Table("video_comment vc"), "vc", query)
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var comments []domain.AdminCommentItem
	offset := (query.PageNo - 1) * query.PageSize
	listDB := r.applyVideoNameFilter(r.db.WithContext(ctx).Table("video_comment vc"), "vc", query)
	err := listDB.
		Select(`
			vc.comment_id,
			vc.p_comment_id,
			vc.video_id,
			COALESCE(vi.video_name, '') AS video_name,
			COALESCE(vi.video_cover, '') AS video_cover,
			vc.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(ui.avatar, '') AS avatar,
			COALESCE(vc.reply_user_id, '') AS reply_user_id,
			COALESCE(reply_user.nick_name, '') AS reply_nick_name,
			COALESCE(vc.content, '') AS content,
			COALESCE(vc.img_path, '') AS img_path,
			COALESCE(DATE_FORMAT(vc.post_time, '%Y-%m-%d %H:%i:%s'), '') AS post_time
		`).
		Joins("LEFT JOIN user_info ui ON vc.user_id = ui.user_id").
		Joins("LEFT JOIN user_info reply_user ON vc.reply_user_id = reply_user.user_id").
		Order("vc.comment_id desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&comments).Error
	if err != nil {
		return nil, 0, err
	}
	return comments, totalCount, nil
}

func (r *InteractRepository) ListDanmuByPage(ctx context.Context, query InteractListQuery) ([]domain.AdminDanmuItem, int64, error) {
	var totalCount int64
	countDB := r.applyVideoNameFilter(r.db.WithContext(ctx).Table("video_danmu vd"), "vd", query)
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var danmuList []domain.AdminDanmuItem
	offset := (query.PageNo - 1) * query.PageSize
	listDB := r.applyVideoNameFilter(r.db.WithContext(ctx).Table("video_danmu vd"), "vd", query)
	err := listDB.
		Select(`
			vd.danmu_id,
			vd.video_id,
			COALESCE(vi.video_name, '') AS video_name,
			vd.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			vd.time,
			COALESCE(vd.text, '') AS text,
			COALESCE(DATE_FORMAT(vd.post_time, '%Y-%m-%d %H:%i:%s'), '') AS post_time
		`).
		Joins("LEFT JOIN user_info ui ON vd.user_id = ui.user_id").
		Order("vd.danmu_id desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&danmuList).Error
	if err != nil {
		return nil, 0, err
	}
	return danmuList, totalCount, nil
}

func (r *InteractRepository) FindCommentDeleteInfo(ctx context.Context, commentID int) (*domain.CommentDeleteInfo, error) {
	var comment domain.CommentDeleteInfo
	err := r.db.WithContext(ctx).
		Table("video_comment").
		Select("comment_id, p_comment_id, video_id").
		Where("comment_id = ?", commentID).
		Take(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *InteractRepository) FindDanmuDeleteInfo(ctx context.Context, danmuID int) (*domain.DanmuDeleteInfo, error) {
	var danmu domain.DanmuDeleteInfo
	err := r.db.WithContext(ctx).
		Table("video_danmu").
		Select("danmu_id, video_id").
		Where("danmu_id = ?", danmuID).
		Take(&danmu).Error
	if err != nil {
		return nil, err
	}
	return &danmu, nil
}

func (r *InteractRepository) VideoExists(ctx context.Context, videoID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("video_info").
		Where("video_id = ?", videoID).
		Count(&count).Error
	return count > 0, err
}

func (r *InteractRepository) DeleteComment(ctx context.Context, comment domain.CommentDeleteInfo) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("DELETE FROM video_comment WHERE comment_id = ?", comment.CommentID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if comment.PCommentID == 0 {
			if err := tx.Table("video_info").
				Where("video_id = ?", comment.VideoID).
				Update("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", 1)).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM video_comment WHERE p_comment_id = ?", comment.CommentID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *InteractRepository) DeleteDanmu(ctx context.Context, danmu domain.DanmuDeleteInfo) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("DELETE FROM video_danmu WHERE danmu_id = ?", danmu.DanmuID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return tx.Table("video_info").
			Where("video_id = ?", danmu.VideoID).
			Update("danmu_count", gorm.Expr("GREATEST(danmu_count - ?, 0)", 1)).Error
	})
}

func (r *InteractRepository) applyVideoNameFilter(db *gorm.DB, tableAlias string, query InteractListQuery) *gorm.DB {
	db = db.Joins("LEFT JOIN video_info vi ON " + tableAlias + ".video_id = vi.video_id")
	if query.VideoNameFuzzy != "" {
		db = db.Where("vi.video_name LIKE ?", "%"+query.VideoNameFuzzy+"%")
	}
	return db
}
